// Purpose: Classify operational failures without recording backend error text.
// Pattern: Safe diagnostic error boundary.
// Usage: Runtime, storage and network paths record these fixed codes.
// Related: internal/diagnostics/handler.go, internal/serverapi/network.go.
package diagnostics

import (
	"bytes"
	"context"
	"log"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"

	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func Code(err error) string {
	return fault.Code(err)
}

type HTTPWriter struct{ Context context.Context }

// Write deliberately classifies net/http diagnostics without forwarding raw panic or TLS text.
func (w HTTPWriter) Write(body []byte) (int, error) {
	code := diagcatalog.HttpServerError
	if bytes.Contains(body, []byte("TLS handshake")) {
		code = diagcatalog.TlsHandshakeFailed
	}
	if bytes.Contains(body, []byte("panic serving")) {
		code = diagcatalog.HttpHandlerPanic
	}
	Log(w.Context, diagcatalog.HTTPServerError, "component", "http", "error_code", code)
	return len(body), nil
}

func HTTPLogger(ctx context.Context) *log.Logger { return log.New(HTTPWriter{Context: ctx}, "", 0) }
