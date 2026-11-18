// Purpose: Read numeric output-budget evidence from a rejected generation HTTP request.
// Pattern: Narrow provider error parser with request correlation.
// Usage: The HTTP generation adapter consults this only after SendOnce rejects a request.
// Related: internal/provider/transport/errors.go, internal/provider/httpclient/request.go.
package transport

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

var rejectedTokens = regexp.MustCompile(`(?i)^(?:this model's maximum context length is ([0-9]{1,10}) tokens[.]\s*(?:however,\s*)?)?you requested ([0-9]{1,10}) output tokens and your prompt contains (at least )?([0-9]{1,10}) input tokens,? for a total of (at least )?([0-9]{1,10}) tokens\b`)

// OutputCapacityError forwards confirmed generation rejections to the context compactor.
func OutputCapacityError(err error, sent int) *modelapi.ContextLimitError {
	var e *responseError
	if !errors.As(err, &e) || (e.status != http.StatusBadRequest && e.status != http.StatusUnprocessableEntity) {
		return nil
	}
	switch strings.ToLower(e.code) {
	case "", "400", "422", "badrequesterror", "bad_request", "invalid_request_error", "context_length_exceeded", "context_window_exceeded", "max_context_length_exceeded":
	default:
		return nil
	}
	match := rejectedTokens.FindStringSubmatch(strings.TrimSpace(e.message))
	if match == nil {
		return nil
	}
	number := func(s string) int {
		n, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return 0
		}
		return int(n)
	}
	requested, input, total := number(match[2]), number(match[4]), number(match[6])
	if requested <= 0 || requested != sent || input <= 0 || int64(total) != int64(requested)+int64(input) || (match[3] == "") != (match[5] == "") {
		return nil
	}
	capacity := number(match[1])
	if match[1] != "" && (capacity == 0 || total <= capacity) {
		return nil
	}
	failure := &modelapi.ContextLimitError{Capacity: capacity, RequestedOutput: sent, Cause: err}
	if match[3] == "" {
		failure.Input = input
	}
	return failure
}
