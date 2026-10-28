package agentclient

import (
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func newTestConn(t *testing.T) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient(
		"passthrough:///agentclient-test",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return conn
}

func TestIsConnHealthy_ShutdownOnly(t *testing.T) {
	conn := newTestConn(t)
	if !isConnHealthy(conn) {
		t.Fatal("fresh conn should be healthy (Idle/Connecting are OK)")
	}
	_ = conn.Close()
	// Give the state machine a tick to observe Shutdown.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !isConnHealthy(conn) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("closed conn should be unhealthy, state=%v", conn.GetState())
}

func TestConnEntry_SoftCloseWaitsForRefs(t *testing.T) {
	conn := newTestConn(t)
	entry := &connEntry{conn: conn}
	entry.acquire()
	entry.acquire()
	entry.closing.Store(true)

	entry.release()
	if !isConnHealthy(conn) {
		t.Fatal("conn must stay open while refs remain")
	}

	entry.release()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !isConnHealthy(conn) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("conn should close after last release, state=%v", conn.GetState())
}

func TestConnPool_EvictIdleSkipsInUse(t *testing.T) {
	pool := &connPool{
		conns: make(map[string]*connEntry),
		ttl:   time.Nanosecond,
		done:  make(chan struct{}),
	}
	conn := newTestConn(t)
	entry := &connEntry{conn: conn}
	entry.acquire()
	entry.lastUsed.Store(time.Now().Add(-time.Hour).UnixNano())
	pool.conns["127.0.0.1:50052"] = entry

	pool.evictIdle()

	if _, ok := pool.conns["127.0.0.1:50052"]; ok {
		t.Fatal("in-use idle entry should be removed from map (soft-evict)")
	}
	if !entry.closing.Load() {
		t.Fatal("in-use entry should be marked closing")
	}
	if !isConnHealthy(conn) {
		t.Fatal("conn must not Close while refs > 0")
	}

	entry.release()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !isConnHealthy(conn) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("conn should close after release, state=%v", conn.GetState())
}

func TestConnPool_ConcurrentAcquireRelease(t *testing.T) {
	pool := &connPool{
		conns: make(map[string]*connEntry),
		ttl:   time.Hour,
		done:  make(chan struct{}),
	}
	conn := newTestConn(t)
	entry := &connEntry{conn: conn}
	pool.conns["127.0.0.1:50052"] = entry

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				lease, err := pool.acquire("127.0.0.1", 50052)
				if err != nil {
					t.Errorf("acquire: %v", err)
					return
				}
				lease.release()
			}
		}()
	}
	wg.Wait()

	if entry.refs.Load() != 0 {
		t.Fatalf("expected refs=0, got %d", entry.refs.Load())
	}
	_ = conn.Close()
}
