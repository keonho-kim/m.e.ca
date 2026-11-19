//go:build !darwin

// Purpose: Use Go's certificate verification against the host trust store on Linux.
// Pattern: Build-selected TLS transport policy.
// Usage: NewClient configures the pinned transport before its first request.
// Related: internal/tools/browser/webfetch/client.go, internal/tools/browser/webfetch/trust_darwin.go.
package webfetch

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"runtime"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func configureOfflineTLS(transport *http.Transport) {
	if runtime.GOOS != "linux" {
		transport.DialTLSContext = func(context.Context, string, string) (net.Conn, error) {
			return nil, fault.New(diag.Unavailable, diag.WebFetchTLSUnavailable)
		}
		return
	}
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
}
