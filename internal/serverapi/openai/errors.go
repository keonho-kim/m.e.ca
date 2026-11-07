// Purpose: Encode OpenAI-shaped client errors without exposing internal causes or credentials.
// Pattern: HTTP error boundary.
// Usage: Compatible handlers and the shared transport security boundary call WriteError.
// Related: internal/serverapi/security.go, internal/serverapi/openai/handler.go.
package openai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/storage/capacity"
	"github.com/keonho-kim/meca/internal/storage/history"
)

type apiError struct {
	Message string  `json:"message"`
	Type    string  `json:"type"`
	Param   *string `json:"param"`
	Code    string  `json:"code"`
}
type inputError struct{ param, message string }

func (e *inputError) Error() string       { return e.message }
func invalid(param, message string) error { return &inputError{param, message} }

func IsPath(path string) bool {
	return path == "/v1/models" || strings.HasPrefix(path, "/v1/models/") || path == "/v1/chat/completions" || path == "/v1/responses" || strings.HasPrefix(path, "/v1/responses/")
}

func WriteError(w http.ResponseWriter, status int, message, code, param string) {
	value := apiError{Message: message, Type: "invalid_request_error", Code: code}
	if param != "" {
		value.Param = &param
	}
	if status >= 500 {
		value.Type = "server_error"
	}
	if status == 401 {
		value.Type = "authentication_error"
	}
	if status == 429 {
		value.Type = "rate_limit_error"
	}
	writeJSON(w, status, map[string]any{"error": value})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	if errors.Is(err, history.ErrReadLimit) || errors.Is(err, capacity.ErrLimit) {
		WriteError(w, 429, err.Error(), "resource_exhausted", "")
		return
	}
	var input *inputError
	if errors.As(err, &input) {
		WriteError(w, 400, input.message, "invalid_parameter", input.param)
		return
	}
	var public *server.Error
	if errors.As(err, &public) {
		status := map[string]int{"invalid_argument": 400, "not_found": 404, "conflict": 409, "failed_precondition": 409, "resource_exhausted": 429, "unavailable": 503}[public.Code]
		if status == 0 {
			status = 500
		}
		WriteError(w, status, h.redact(public.Message), public.Code, "")
		return
	}
	if errors.Is(err, context.Canceled) {
		WriteError(w, 409, "observer disconnected; the accepted task continues", "observer_cancelled", "")
		return
	}
	WriteError(w, 500, "could not complete the compatible API request; inspect the response before retrying", "server_error", "")
}
