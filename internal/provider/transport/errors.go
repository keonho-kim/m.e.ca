// Purpose: Interpret bounded provider error details and redact them before presentation.
// Pattern: HTTP error translation.
// Usage: Transport and stream recovery return explicit provider failure causes.
// Related: internal/provider/transport/transport.go, internal/provider/transport/retry.go.
package transport

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

type responseError struct {
	status        int
	code, message string
	param         string
	retryAfter    time.Duration
}

func (e *responseError) Error() string {
	label := diagcatalog.Text(diagcatalog.ProviderGenerationFailure)
	if e.status != 0 {
		label = diagcatalog.Text(diagcatalog.ProviderHTTPFailure, e.status, http.StatusText(e.status))
	}
	if e.code != "" {
		label = diagcatalog.Text(diagcatalog.ProviderFailureCode, label, e.code)
	}
	if e.message != "" {
		label = diagcatalog.Text(diagcatalog.ProviderFailureDetail, label, e.message)
	}
	return label
}

func ParseError(status int, body []byte, after string) error {
	e := &responseError{status: status}
	var envelope struct {
		Error   json.RawMessage `json:"error"`
		Message string          `json:"message"`
		Detail  json.RawMessage `json:"detail"`
		Code    json.RawMessage `json:"code"`
	}
	if json.Unmarshal(body, &envelope) == nil {
		if len(envelope.Error) == 0 {
			e.message = envelope.Message
			if e.message == "" {
				if err := json.Unmarshal(envelope.Detail, &e.message); err != nil {
					e.message = ""
				}
			}
		}
		if len(envelope.Error) > 0 && json.Unmarshal(envelope.Error, &e.message) != nil {
			var detail struct {
				Message             string
				Code                json.RawMessage
				Type, Status, Param string
			}
			if json.Unmarshal(envelope.Error, &detail) == nil {
				e.message = detail.Message
				e.param = detail.Param
				if json.Unmarshal(detail.Code, &e.code) != nil {
					e.code = detail.Status
					var numeric int
					if status == 0 && json.Unmarshal(detail.Code, &numeric) == nil && numeric >= 400 && numeric <= 599 {
						e.status = numeric
					}
				}
				if e.code == "" {
					e.code = detail.Type
				}
			}
		}
	}
	if e.code == "" {
		_ = json.Unmarshal(envelope.Code, &e.code)
	}
	if len(e.message) > 8<<10 {
		e.message = diagcatalog.Text(diagcatalog.ProviderDetailOmitted)
	}
	if len(e.code) > 128 {
		e.code = ""
	}
	if len(e.param) > 256 {
		e.param = ""
	}

	if e.status == 0 {
		switch e.code {
		case "invalid_request_error", "invalid_api_key", "unsupported_parameter":
			e.status = http.StatusBadRequest
		case "authentication_error":
			e.status = http.StatusUnauthorized
		case "permission_error":
			e.status = http.StatusForbidden
		case "not_found_error":
			e.status = http.StatusNotFound
		case "overloaded_error":
			e.status = 529
		}
	}
	if seconds, err := strconv.Atoi(after); err == nil && seconds > 0 {
		e.retryAfter = time.Duration(min(seconds, 30)) * time.Second
	} else if date, err := http.ParseTime(after); err == nil {
		e.retryAfter = min(30*time.Second, max(0, time.Until(date)))
	}
	switch e.code {
	case "context_length_exceeded", "context_window_exceeded", "max_context_length_exceeded":
		return &modelapi.ContextLimitError{Cause: e}
	}
	return e
}

func (c *Client) FailureReason(err error) string {
	text := err.Error()
	if c.key != "" {
		text = strings.ReplaceAll(text, c.key, "[redacted]")
	}
	var result strings.Builder
	count := 0
	for _, r := range text {
		if count == 500 {
			result.WriteRune('…')
			break
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			r = ' '
		}
		result.WriteRune(r)
		count++
	}
	return result.String()
}

func (c *Client) Redact(text string) string {
	if c.key != "" {
		return strings.ReplaceAll(text, c.key, "[REDACTED]")
	}
	return text
}
