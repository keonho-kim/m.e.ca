// Purpose: Bound accepted TCP connections before HTTP parsing or TLS allocation.
// Pattern: Listener adapter with connection-owned admission release.
// Usage: ListenAndServe wraps its listener once; every close returns the connection slot.
// Related: internal/serverapi/network.go, internal/serverapi/transport.go.
package serverapi

import (
	"net"
	"sync"
)

const maxConnections = 64

type connectionListener struct {
	net.Listener
	slots chan struct{}
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
