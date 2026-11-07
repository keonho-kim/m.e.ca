// Purpose: Reject excess TCP connections before HTTP allocation and release admission exactly once.
// Pattern: Listener-boundary lifecycle tests.
// Usage: go test -race ./internal/serverapi -run TestConnectionAdmission.
// Related: internal/serverapi/connections.go, internal/serverapi/network_test.go.
package serverapi

import (
	"errors"
	"net"
	"sync"
	"testing"
)

type connectionSequence struct {
	net.Listener
	values []net.Conn
}

func (l *connectionSequence) Accept() (net.Conn, error) {
	if len(l.values) == 0 {
		return nil, net.ErrClosed
	}
	c := l.values[0]
	l.values = l.values[1:]
	return c, nil
}

func TestConnectionAdmissionClosesOverflowAndReleasesOnce(t *testing.T) {
	first, firstPeer := net.Pipe()
	defer firstPeer.Close()
	overflow, overflowPeer := net.Pipe()
	defer overflowPeer.Close()
	base := &connectionSequence{values: []net.Conn{first, overflow}}
	listener := &connectionListener{Listener: base, slots: make(chan struct{}, 1)}
	owned, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := listener.Accept(); !errors.Is(err, net.ErrClosed) {
		t.Fatal(err)
	}
	if _, err := overflowPeer.Write([]byte("closed")); err == nil {
		t.Fatal("overflow connection remained open")
	}
	var closers sync.WaitGroup
	for range 4 {
		closers.Go(func() {
			if err := owned.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	closers.Wait()
	if len(listener.slots) != 0 {
		t.Fatal("connection admission was not released")
	}
	next, peer := net.Pipe()
	defer peer.Close()
	base.values = []net.Conn{next}
	accepted, err := listener.Accept()
	if err != nil {
		t.Fatal("released connection slot was not reused", err)
	}
	accepted.Close()
}
