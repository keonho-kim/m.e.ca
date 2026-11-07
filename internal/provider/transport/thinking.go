// Purpose: Recognize explicit provider rejection of an automatically requested thought summary.
// Pattern: Provider capability-error interpretation.
// Usage: HTTP generation removes only its implicit summary option after a rejected request.
// Related: internal/provider/transport/errors.go, internal/provider/httpclient/request.go.
package transport

import (
	"errors"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"strings"
)

func ThinkingSummaryUnsupported(err error, kind connection.ProviderKind) bool {
	var response *responseError
	if !errors.As(err, &response) || response.status != 400 {
		return false
	}
	switch kind {
	case connection.OpenAI:
		return response.code == "unsupported_parameter" && (response.param == "reasoning.summary" || response.param == "reasoning")
	case connection.Gemini:
		message := strings.ToLower(response.message)
		return response.code == "INVALID_ARGUMENT" && strings.Contains(message, "thinking") && (strings.Contains(message, "not supported") || strings.Contains(message, "unsupported"))
	}
	return false
}
