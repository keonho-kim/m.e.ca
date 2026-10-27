//go:build linux

// Purpose: Detect DBus TCP or nonce transmission through the real credential adapter and preserve Unix dialing.
// Pattern: Isolated subprocess transport regression tests.
// Usage: GOFLAGS='-tags=godbus_no_tcp' go test ./internal/config/credentials -run TestCredentialDBus.
// Related: internal/config/credentials/keyring.go, .goreleaser.yaml, .github/workflows/release.yml.
package credentials

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestCredentialDBusTransport(t *testing.T) {
	for _, network := range []string{"tcp", "nonce-tcp", "unix"} {
		t.Run(network, func(t *testing.T) {
			nonce := []byte("meca-transport-test-canary")
			address := "127.0.0.1:0"
			protocol := "tcp4"
			if network == "unix" {
				protocol, address = "unix", filepath.Join(t.TempDir(), "dbus.sock")
			}
			listener, err := net.Listen(protocol, address)
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			if network == "unix" {
				address = "unix:path=" + address
			} else {
				_, port, err := net.SplitHostPort(listener.Addr().String())
				if err != nil {
					t.Fatal(err)
				}
				address = fmt.Sprintf("%s:host=127.0.0.1,port=%s,family=ipv4", network, port)
				if network == "nonce-tcp" {
					path := filepath.Join(t.TempDir(), "nonce")
					if err := os.WriteFile(path, nonce, 0600); err != nil {
						t.Fatal(err)
					}
					address += ",noncefile=" + path
				}
			}
			type observation struct {
				connected bool
				data      string
				err       error
			}
			accepted := make(chan observation, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					accepted <- observation{err: err}
					return
				}
				result := observation{connected: true}
				if network == "nonce-tcp" {
					if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
						result.err = err
					} else {
						body := make([]byte, len(nonce))
						n, readErr := io.ReadFull(conn, body)
						result.data, result.err = string(body[:n]), readErr
					}
				}
				conn.Close()
				accepted <- result
			}()
			err = credentialDBusProcess(t, address, "probe")
			listener.Close()
			result := <-accepted // Join the accept/read worker even if the subprocess failed.
			if err != nil {
				t.Fatal(err)
			}
			if result.err != nil && !errors.Is(result.err, net.ErrClosed) {
				t.Fatal(result.err)
			}
			if result.connected != (network == "unix") || result.data != "" {
				t.Fatalf("%s: connected=%t, transmitted=%q; only Unix dialing is permitted", network, result.connected, result.data)
			}
		})
	}
}

func credentialDBusProcess(t *testing.T, address, mode string) error {
	t.Helper()
	binary, err := os.Executable()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, "-test.run=^TestCredentialDBusHelper$", "-test.v")
	command.Env = append(os.Environ(), "DBUS_SESSION_BUS_ADDRESS="+address, "MECA_DBUS_TEST_HELPER="+mode)
	command.WaitDelay = time.Second
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("credential subprocess: %w\n%s", err, output)
	}
	return nil
}

func TestCredentialDBusHelper(t *testing.T) {
	switch os.Getenv("MECA_DBUS_TEST_HELPER") {
	case "probe":
		if _, err := (&Vault{}).Get("transport-test-no-credential"); err == nil {
			t.Fatal("fixture connection unexpectedly returned a credential")
		}
	case "secret-service":
		credentialSecretServiceRoundTrip(t)
	}
}
