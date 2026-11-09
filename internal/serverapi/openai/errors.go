// Purpose: Encode OpenAI-shaped client errors without exposing internal causes or credentials.
// Pattern: HTTP error boundary.
// Usage: Compatible handlers and the shared transport security boundary call WriteError.
// Related: internal/serverapi/security.go, internal/serverapi/openai/handler.go.
package openai

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type apiError struct {
	Message string  `json:"message"`
	Type    string  `json:"type"`
	Param   *string `json:"param"`
	Code    string  `json:"code"`
}

func IsPath(path string) bool {
	return path == "/v1/models" || strings.HasPrefix(path, "/v1/models/") || path == "/v1/chat/completions" || path == "/v1/responses" || strings.HasPrefix(path, "/v1/responses/")
}

func WriteError(w http.ResponseWriter, status int, message, code, param string) {
	value := apiError{Message: message, Type: diagcatalog.CompatibleType(status), Code: code}
	if param != "" {
		value.Param = &param
	}
	writeJSON(w, status, map[string]any{"error": value})
}

func writeJSON(w http.ResponseWriter, status int, value any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(value)
	if reporter, ok := w.(interface{ ReportError(error) }); ok {
		reporter.ReportError(err)
	}
	if err != nil {
		return faults.New(diagcatalog.DeliveryFailed, diagcatalog.WrappedCause, err)
	}
	return err
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	code := faults.Code(err)
	var input *faults.InputError
	parameter := errors.As(err, &input) && code == diagcatalog.InvalidArgument
	param := ""
	if parameter {
		param = input.Param
	}
	WriteError(w, diagcatalog.HTTPStatus(code), faults.PublicMessage(err, h.Redact), diagcatalog.CompatibleCode(code, parameter), param)
}
