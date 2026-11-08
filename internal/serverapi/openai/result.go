// Purpose: Hold original-read admission through masking and final output serialization.
// Pattern: Request-owned result lifetime with shared process admission.
// Usage: JSON responses and terminal SSE events use the same bounded result path.
// Related: internal/serverapi/openai/handler.go, internal/serverapi/eventstream/await.go.
package openai

import (
	"context"
	"net/http"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/processmemory"
	"github.com/keonho-kim/meca/internal/serverapi/admission"
	"github.com/keonho-kim/meca/internal/serverapi/eventstream"
)

func (h *Handler) withResult(ctx context.Context, writer *eventstream.Writer, value server.Response, send func(string) error) error {
	h.readOnce.Do(func() {
		if h.Reads == nil {
			h.Reads = admission.New(1)
		}
	})
	release, err := eventstream.Await(ctx, writer, func(ctx context.Context) (func(), error) {
		releaseRead, err := h.Reads.Acquire(ctx, 1)
		if err != nil {
			return nil, err
		}
		releaseMemory, err := processmemory.AcquirePayload(ctx, server.MaxOriginalBytes)
		if err != nil {
			releaseRead()
			return nil, err
		}
		return func() { releaseMemory(); releaseRead() }, nil
	})
	if release != nil {
		defer release()
	}
	if err != nil {
		return err
	}
	text, err := eventstream.Await(ctx, writer, func(ctx context.Context) (string, error) {
		text, err := h.Service.ResponseResult(ctx, value)
		if err != nil {
			return "", err
		}
		return h.redact(text), nil
	})
	if err != nil {
		return err
	}
	return send(text)
}

func (h *Handler) writeResult(w http.ResponseWriter, r *http.Request, value server.Response, chat bool) {
	send := func(text string) error {
		if chat {
			return writeJSON(w, 200, chatValue(value, text))
		} else {
			return writeJSON(w, 200, responseValue(value, text))
		}
	}
	if !terminal(value) {
		_ = send("")
		return
	}
	if err := h.withResult(r.Context(), nil, value, send); err != nil {
		if faults.Code(err) != diagcatalog.DeliveryFailed {
			h.fail(w, err)
		}
	}
}
