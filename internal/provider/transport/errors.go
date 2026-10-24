// Purpose: Interpret bounded provider error details and redact them before presentation.
// Pattern: HTTP error translation.
// Usage: Transport and stream recovery return explicit provider failure causes.
// Related: internal/provider/transport/transport.go, internal/provider/transport/retry.go.
package transport

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

type responseError struct {
	status        int
	code, message string
	retryAfter    time.Duration
}

func (e *responseError) Error() string {
	label := "provider generation error"
	if e.status != 0 {
		label = fmt.Sprintf("HTTP %d (%s)", e.status, http.StatusText(e.status))
	}
	if e.code != "" {
		label += " [" + e.code + "]"
	}
	if e.message != "" {
		label += ": " + e.message
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
				Message string
				Code    json.RawMessage
			}
			if json.Unmarshal(envelope.Error, &detail) == nil {
				e.message = detail.Message
				if json.Unmarshal(detail.Code, &e.code) != nil {
					e.code = ""
				}
			}
		}
	}
	if e.code == "" {
		_ = json.Unmarshal(envelope.Code, &e.code)
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
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return ' '
		}
		return r
	}, text)
	runes := []rune(text)
	if len(runes) > 500 {
		text = string(runes[:500]) + "…"
	}
	return text
}

func (c *Client) Redact(text string) string {
	if c.key != "" {
		return strings.ReplaceAll(text, c.key, "[REDACTED]")
	}
	return text
}
