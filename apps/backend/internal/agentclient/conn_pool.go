package agentclient

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	_ "google.golang.org/grpc/encoding/gzip"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
)

const (
	defaultPoolTTL      = 5 * time.Minute
	poolCleanupInterval = 1 * time.Minute
	clientKeepaliveTime = 30 * time.Second
	clientKeepaliveWait = 10 * time.Second
	minConnectTimeout   = 20 * time.Second
)

type connEntry struct {
	conn     *grpc.ClientConn
	lastUsed atomic.Int64
	refs     atomic.Int32
	closing  atomic.Bool
}

func (e *connEntry) touch() {
	e.lastUsed.Store(time.Now().UnixNano())
}

func (e *connEntry) idleSince() time.Duration {
	last := time.Unix(0, e.lastUsed.Load())
	return time.Since(last)
}

func (e *connEntry) acquire() {
	e.refs.Add(1)
	e.touch()
}

func (e *connEntry) release() {
	n := e.refs.Add(-1)
	if n < 0 {
		e.refs.Store(0)
		return
	}
	if n == 0 && e.closing.Load() {
		_ = e.conn.Close()
	}
}

// connLease holds a pooled connection reference until release is called.
type connLease struct {
	entry *connEntry
	conn  *grpc.ClientConn
}

func (l *connLease) release() {
	if l == nil || l.entry == nil {
		return
	}
	l.entry.release()
	l.entry = nil
}

type connPool struct {
	mu                 sync.RWMutex
	conns              map[string]*connEntry
	rootCA             []byte
	clientCert         *tls.Certificate
	insecureSkipVerify bool
	ttl                time.Duration
	done               chan struct{}
}

func newConnPool(rootCA []byte, clientCert *tls.Certificate, insecureSkipVerify bool) *connPool {
	p := &connPool{
		conns:              make(map[string]*connEntry),
		rootCA:             rootCA,
		clientCert:         clientCert,
		insecureSkipVerify: insecureSkipVerify,
		ttl:                defaultPoolTTL,
		done:               make(chan struct{}),
	}
	go p.cleanupLoop()
	return p
}

func (p *connPool) acquire(host string, port int) (*connLease, error) {
	conn, entry, err := p.getOrDialConn(host, port)
	if err != nil {
		return nil, err
	}
	return &connLease{entry: entry, conn: conn}, nil
}

func (p *connPool) getOrDial(host string, port int) (pb.AgentServiceClient, *connLease, error) {
	lease, err := p.acquire(host, port)
	if err != nil {
		return nil, nil, err
	}
	return pb.NewAgentServiceClient(lease.conn), lease, nil
}

func (p *connPool) dial(host, addr string) (*grpc.ClientConn, error) {
	tlsConfig := &tls.Config{
		ServerName:         host,
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: p.insecureSkipVerify,
	}
	if !p.insecureSkipVerify {
		roots := x509.NewCertPool()
		if ok := roots.AppendCertsFromPEM(p.rootCA); !ok {
			return nil, fmt.Errorf("invalid CA cert")
		}
		tlsConfig.RootCAs = roots
	}
	if p.clientCert != nil {
		tlsConfig.Certificates = []tls.Certificate{*p.clientCert}
	}

	return grpc.NewClient(addr,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                clientKeepaliveTime,
			Timeout:             clientKeepaliveWait,
			PermitWithoutStream: true,
		}),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  1 * time.Second,
				Multiplier: 1.6,
				Jitter:     0.2,
				MaxDelay:   30 * time.Second,
			},
			MinConnectTimeout: minConnectTimeout,
		}),
		grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")),
		grpc.WithUnaryInterceptor(retryUnaryInterceptor()),
	)
}

func (p *connPool) cleanupLoop() {
	ticker := time.NewTicker(poolCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.done:
			return
		case <-ticker.C:
			p.evictIdle()
		}
	}
}

func (p *connPool) evictIdle() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for addr, entry := range p.conns {
		idleTooLong := entry.idleSince() > p.ttl
		dead := !isConnHealthy(entry.conn)
		if !idleTooLong && !dead {
			continue
		}
		p.removeLocked(addr, entry)
	}
}

// removeLocked drops the entry from the map. If refs remain, the conn is
// soft-closed and actually closed on the last release; otherwise Close now.
func (p *connPool) removeLocked(addr string, entry *connEntry) {
	delete(p.conns, addr)
	if entry.refs.Load() > 0 {
		entry.closing.Store(true)
		return
	}
	_ = entry.conn.Close()
}

func (p *connPool) Close() {
	close(p.done)

	p.mu.Lock()
	defer p.mu.Unlock()

	for addr, entry := range p.conns {
		p.removeLocked(addr, entry)
	}
}

func (p *connPool) getOrDialConn(host string, port int) (*grpc.ClientConn, *connEntry, error) {
	if port == 0 {
		port = defaultAgentPort
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	p.mu.RLock()
	entry, ok := p.conns[addr]
	if ok && isConnHealthy(entry.conn) && !entry.closing.Load() {
		entry.acquire()
		p.mu.RUnlock()
		return entry.conn, entry, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	if entry, ok = p.conns[addr]; ok && isConnHealthy(entry.conn) && !entry.closing.Load() {
		entry.acquire()
		return entry.conn, entry, nil
	}

	if entry != nil {
		p.removeLocked(addr, entry)
	}

	conn, err := p.dial(host, addr)
	if err != nil {
		return nil, nil, err
	}

	entry = &connEntry{conn: conn}
	entry.acquire()
	p.conns[addr] = entry
	return conn, entry, nil
}

// isConnHealthy reports whether the conn may still be used. TransientFailure
// is recoverable by gRPC's internal reconnect — do not treat it as dead.
func isConnHealthy(conn *grpc.ClientConn) bool {
	return conn.GetState() != connectivity.Shutdown
}
