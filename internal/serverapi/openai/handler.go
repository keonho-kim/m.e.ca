// Purpose: Mount optional compatible HTTP endpoints and detach observers from engine task lifetime.
// Pattern: Concrete HTTP adapter over the server application service.
// Usage: Transport mounts Handler on the same authenticated listener as its native protocol.
// Related: internal/serverapi/openai/input.go, internal/serverapi/openai/stream.go.
package openai

import (
	"net/http"
	"strings"
	"sync"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/serverapi/admission"
)

type Handler struct {
	Service  *server.Service
	Redact   func(string) string
	Streams  chan struct{}
	Reads    *admission.Budget
	readOnce sync.Once
}

func (h *Handler) Mount(mux *http.ServeMux) {
	mux.HandleFunc("/v1/models", h.models)
	mux.HandleFunc("/v1/models/", h.models)
	mux.HandleFunc("/v1/chat/completions", h.chat)
	mux.HandleFunc("/v1/responses", h.responses)
	mux.HandleFunc("/v1/responses/", h.response)
}
func (h *Handler) redact(value string) string {
	if h.Redact != nil {
		return h.Redact(value)
	}
	return value
}
func method(w http.ResponseWriter, r *http.Request, want string) bool {
	if r.Method == want {
		return true
	}
	w.Header().Set("Allow", want)
	WriteError(w, 405, diagcatalog.Text(diagcatalog.OpenaiMethodIsNotSupported), diagcatalog.MethodNotAllowed, "")
	return false
}
func (h *Handler) models(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET") {
		return
	}
	model := map[string]any{"id": "meca", "object": "model", "created": 0, "owned_by": "meca"}
	switch r.URL.Path {
	case "/v1/models":
		writeJSON(w, 200, map[string]any{"object": "list", "data": []any{model}})
	case "/v1/models/meca":
		writeJSON(w, 200, model)
	default:
		WriteError(w, 404, diagcatalog.Text(diagcatalog.OpenaiModelWasNotFound), diagcatalog.ModelNotFound, "model")
	}
}

func (h *Handler) submit(w http.ResponseWriter, r *http.Request, kind string) {
	if !method(w, r, "POST") {
		return
	}
	p, err := parse(r, w, kind)
	if err != nil {
		h.fail(w, err)
		return
	}
	if p.stream {
		select {
		case h.Streams <- struct{}{}:
			defer func() { <-h.Streams }()
		default:
			WriteError(w, 429, diagcatalog.Text(diagcatalog.ServerapiTooManyObservers), diagcatalog.ResourceExhausted, "")
			return
		}
	}
	accepted, err := h.Service.Submit(r.Context(), p.input)
	p.input.Messages = nil
	admission.ReleaseInput(r.Context())
	if accepted.Response.ID != "" {
		w.Header().Set("X-Request-ID", accepted.Response.ID)
		w.Header().Set("X-Should-Retry", "false")
	}
	if accepted.Response.Task != "" {
		w.Header().Set("X-Meca-Task-ID", accepted.Response.Task)
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	if p.stream {
		h.stream(w, r, p, accepted)
		return
	}
	if p.input.Background {
		value, err := h.Service.ResponseState(r.Context(), accepted.Response.ID, false)
		if err != nil {
			h.fail(w, err)
			return
		}
		value.Error = h.redact(value.Error)
		h.writeResult(w, r, value, false)
		return
	}
	value, err := h.Service.WaitResponse(r.Context(), accepted.Response.ID)
	if err != nil {
		h.fail(w, err)
		return
	}
	value.Error = h.redact(value.Error)
	if kind == "chat" && value.Status != "completed" && value.Status != "incomplete" {
		h.taskFailure(w, value)
		return
	}
	h.writeResult(w, r, value, kind == "chat")
}

func terminal(value server.Response) bool {
	return value.Status != "queued" && value.Status != "in_progress"
}
func batchCursor(batch server.Batch, current string) string {
	if batch.Snapshot != nil {
		return batch.Snapshot.Cursor
	}
	if len(batch.Events) > 0 {
		return batch.Events[len(batch.Events)-1].Cursor
	}
	return current
}
func (h *Handler) taskFailure(w http.ResponseWriter, value server.Response) {
	code := value.Code
	if code == "" {
		code = diagcatalog.ExecutionFailed
	}
	message := value.Error
	if message == "" {
		message = "the engine task did not complete"
	}
	status := diagcatalog.HTTPStatus(code)
	if value.Status == "cancelled" || code == diagcatalog.ApprovalRequired || code == diagcatalog.Interrupted {
		status = 409
	}
	WriteError(w, status, h.redact(message), diagcatalog.CompatibleCode(code, false), "")
}
func (h *Handler) chat(w http.ResponseWriter, r *http.Request)      { h.submit(w, r, "chat") }
func (h *Handler) responses(w http.ResponseWriter, r *http.Request) { h.submit(w, r, "response") }

func (h *Handler) response(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/responses/"), "/")
	cancel := len(parts) == 2 && parts[1] == "cancel"
	if len(parts) != 1 && !cancel {
		WriteError(w, 404, diagcatalog.Text(diagcatalog.OpenaiRouteWasNotFound), diagcatalog.NotFound, "")
		return
	}
	want := "GET"
	if cancel {
		want = "POST"
	}
	if !method(w, r, want) {
		return
	}
	value, err := h.Service.ResponseState(r.Context(), parts[0], true)
	if err != nil {
		h.fail(w, err)
		return
	}
	if cancel {
		if !value.Background {
			WriteError(w, 409, diagcatalog.Text(diagcatalog.OpenaiOnlyBackgroundResponsesCanBeCancelled), diagcatalog.FailedPrecondition, "")
			return
		}
		if !terminal(value) {
			if err := h.Service.Cancel(value.Task); err != nil {
				latest, cause := h.Service.ResponseState(r.Context(), value.ID, true)
				if cause != nil || !terminal(latest) {
					h.fail(w, err)
					return
				}
			}
			value, err = h.Service.WaitResponse(r.Context(), value.ID)
			if err != nil {
				h.fail(w, err)
				return
			}
		}
	}
	value.Error = h.redact(value.Error)
	h.writeResult(w, r, value, false)
}
