// Purpose: Bound accepted TCP connections and release those waiting for their first request on shutdown.
// Pattern: Listener adapter with connection-owned admission release.
// Usage: ListenAndServe wraps its listener once; every close returns the connection slot.
// Related: internal/serverapi/network.go, internal/serverapi/transport.go.
package serverapi

import (
	"net"
	"net/http"
	"sync"
)

const maxConnections = 64

type connectionListener struct {
	net.Listener
	slots     chan struct{}
	mu        sync.Mutex
	unstarted map[net.Conn]struct{}
	stopping  bool
}

func (l *connectionListener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		select {
		case l.slots <- struct{}{}:
			return &admittedConnection{Conn: conn, release: func() { <-l.slots }}, nil
		default:
			_ = conn.Close()
		}
	}
}

func (l *connectionListener) trackState(conn net.Conn, state http.ConnState) {
	if state == http.StateIdle {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if state != http.StateNew {
		delete(l.unstarted, conn)
		return
	}
	if l.stopping {
		_ = conn.Close()
		return
	}
	if l.unstarted == nil {
		l.unstarted = make(map[net.Conn]struct{})
	}
	l.unstarted[conn] = struct{}{}
}

func (l *connectionListener) closeUnstarted() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stopping = true
	for conn := range l.unstarted {
		_ = conn.Close()
	}
	l.unstarted = nil
}

type admittedConnection struct {
	net.Conn
	once    sync.Once
	err     error
	release func()
}

func (c *admittedConnection) Close() error {
	c.once.Do(func() {
		c.err = c.Conn.Close()
		c.release()
	})
	return c.err
}
