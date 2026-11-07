// Purpose: Keep Secret Service dialing on Unix sockets and cancel an unresponsive authentication peer.
// Pattern: Isolated IPC boundary tests without a real credential daemon.
// Usage: go test ./internal/config/credentials/osstore on Linux.
// Related: internal/config/credentials/osstore/secret_connection_linux.go.
package osstore

import (
	"context"
	"errors"
	"io"
	"net"
	"path/filepath"
	"testing"
)

func TestSecretBusParsesOnlyUnambiguousUnixAddresses(t *testing.T) {
	for _, address := range []string{"tcp:host=127.0.0.1,port=1", "nonce-tcp:host=127.0.0.1,port=1", "unix:path=relative", "unix:path=/a,abstract=b", "unix:path=%00", "unix:path=%invalid"} {
		if _, err := secretBusPath(address); err == nil {
			t.Fatal("unsafe address accepted", address)
		}
	}
	if path, err := secretBusPath("unix:path=/tmp/space%20name,guid=abc"); err != nil || path != "/tmp/space name" {
		t.Fatal(path, err)
	}
}

func TestSecretBusCancellationClosesAuthentication(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bus")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+path)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	result := make(chan error, 1)
	go func() { _, err := (Store{Service: "meca-test"}).Get(ctx, "fixture"); result <- err }()
	peer, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	var initial [1]byte
	if _, err := io.ReadFull(peer, initial[:]); err != nil {
		t.Fatal(err)
	}
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatal("authentication was not canceled", err)
	}
	if _, err := io.Copy(io.Discard, peer); err != nil {
		t.Fatal(err)
	}
}
