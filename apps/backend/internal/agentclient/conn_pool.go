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
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	_ "google.golang.org/grpc/encoding/gzip"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
)

const (
	defaultPoolTTL      = 5 * time.Minute
	poolCleanupInterval = 1 * time.Minute
)

type connEntry struct {
	conn     *grpc.ClientConn
	lastUsed atomic.Int64
}

func (e *connEntry) touch() {
	e.lastUsed.Store(time.Now().UnixNano())
}

func (e *connEntry) idleSince() time.Duration {
	last := time.Unix(0, e.lastUsed.Load())
	return time.Since(last)
}

type connPool struct {
	mu                 sync.RWMutex
	conns              map[string]*connEntry
	rootCA             []byte
	insecureSkipVerify bool
	ttl                time.Duration
	done               chan struct{}
}

func newConnPool(rootCA []byte, insecureSkipVerify bool) *connPool {
	p := &connPool{
		conns:              make(map[string]*connEntry),
		rootCA:             rootCA,
		insecureSkipVerify: insecureSkipVerify,
		ttl:                defaultPoolTTL,
		done:               make(chan struct{}),
	}
	go p.cleanupLoop()
	return p
}

func (p *connPool) getOrDial(host string, port int) (pb.AgentServiceClient, error) {
	conn, err := p.getOrDialConn(host, port)
	if err != nil {
		return nil, err
	}
	return pb.NewAgentServiceClient(conn), nil
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

	return grpc.NewClient(addr,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             5 * time.Second,
			PermitWithoutStream: true,
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
		if entry.idleSince() > p.ttl || !isConnHealthy(entry.conn) {
			entry.conn.Close()
			delete(p.conns, addr)
		}
	}
}

func (p *connPool) Close() {
	close(p.done)

	p.mu.Lock()
	defer p.mu.Unlock()

	for addr, entry := range p.conns {
		entry.conn.Close()
		delete(p.conns, addr)
	}
}

func (p *connPool) getOrDialConn(host string, port int) (*grpc.ClientConn, error) {
	if port == 0 {
		port = defaultAgentPort
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	p.mu.RLock()
	entry, ok := p.conns[addr]
	p.mu.RUnlock()

	if ok && isConnHealthy(entry.conn) {
		entry.touch()
		return entry.conn, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if entry, ok = p.conns[addr]; ok && isConnHealthy(entry.conn) {
		entry.touch()
		return entry.conn, nil
	}

	if entry != nil {
		entry.conn.Close()
		delete(p.conns, addr)
	}

	conn, err := p.dial(host, addr)
	if err != nil {
		return nil, err
	}

	entry = &connEntry{conn: conn}
	entry.touch()
	p.conns[addr] = entry
	return conn, nil
}

func isConnHealthy(conn *grpc.ClientConn) bool {
	state := conn.GetState()
	return state != connectivity.Shutdown && state != connectivity.TransientFailure
}
