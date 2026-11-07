// Purpose: Open a context-owned private Unix DBus connection without autolaunch or network fallback.
// Pattern: Explicit IPC transport boundary.
// Usage: Each Secret Service operation authenticates one connection and closes it afterward.
// Related: internal/config/credentials/osstore/store_linux.go, internal/config/credentials/dbus_linux_test.go.
package osstore

import (
	"context"
	"errors"
	"fmt"
	dbus "github.com/godbus/dbus/v5"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func connectSecretBus(ctx context.Context) (*dbus.Conn, error) {
	address := os.Getenv("DBUS_SESSION_BUS_ADDRESS")
	if address == "" {
		directory := os.Getenv("XDG_RUNTIME_DIR")
		if directory == "" {
			directory = filepath.Join("/run/user", fmt.Sprint(os.Getuid()))
		}
		address = "unix:path=" + filepath.Join(directory, "bus")
	}
	if len(address) > 4096 {
		return nil, fmt.Errorf("DBus address exceeds limit")
	}
	var failures error
	for _, entry := range strings.Split(address, ";") {
		path, err := secretBusPath(entry)
		if err != nil {
			failures = errors.Join(failures, err)
			continue
		}
		raw, err := (&net.Dialer{}).DialContext(ctx, "unix", path)
		if err != nil {
			failures = errors.Join(failures, err)
			continue
		}
		conn, err := dbus.NewConn(raw)
		if err != nil {
			_ = raw.Close()
			return nil, err
		}
		// Authentication has no context API. Closing its owned connection aborts
		// the handshake, and this callback is joined before ownership transfers.
		done := make(chan struct{})
		stop := context.AfterFunc(ctx, func() { _ = conn.Close(); close(done) })
		err = conn.Auth(nil)
		if err == nil {
			err = conn.Hello()
		}
		if !stop() {
			<-done
		}
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		return conn, nil
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return nil, fmt.Errorf("OS credentials require a running Unix session bus: %w", failures)
}

func secretBusPath(address string) (string, error) {
	if !strings.HasPrefix(address, "unix:") {
		return "", fmt.Errorf("only Unix DBus transports are supported")
	}
	path := ""
	for _, field := range strings.Split(strings.TrimPrefix(address, "unix:"), ",") {
		name, value, ok := strings.Cut(field, "=")
		if !ok {
			return "", fmt.Errorf("invalid Unix DBus address")
		}
		if name != "path" && name != "abstract" {
			continue
		}
		if path != "" {
			return "", fmt.Errorf("ambiguous Unix DBus address")
		}
		decoded, err := url.PathUnescape(value)
		if err != nil || decoded == "" || strings.ContainsRune(decoded, '\x00') {
			return "", fmt.Errorf("invalid Unix DBus socket")
		}
		path = decoded
		if name == "abstract" {
			path = "\x00" + path
		} else if !filepath.IsAbs(path) {
			return "", fmt.Errorf("DBus socket path must be absolute")
		}
	}
	if path == "" {
		return "", fmt.Errorf("unix DBus socket is missing")
	}
	return path, nil
}
