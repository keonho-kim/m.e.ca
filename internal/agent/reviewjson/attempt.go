// Purpose: Bound raw review diagnostics without losing the indication of omitted text.
// Pattern: Pure diagnostic projection.
// Usage: Review use cases publish one Attempt per logical generation through Observe.
// Related: internal/agent/run/review_diagnostic.go, internal/agent/reviewjson/object.go.
package reviewjson

import "unicode/utf8"

const MaxDiagnosticBytes = 16 * 1024 * 1024

type Attempt struct {
	Role      string `json:"role"`
	Number    int    `json:"attempt"`
	Raw       string `json:"response"`
	Error     string `json:"error,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}
type Observe func(Attempt) error

func NewAttempt(role string, number int, raw string, err error) Attempt {
	a := Attempt{Role: role, Number: number, Raw: raw}
	if err != nil {
		a.Error = err.Error()
	}
	if len(a.Raw) > MaxDiagnosticBytes {
		end := MaxDiagnosticBytes
		for end > 0 && !utf8.RuneStart(a.Raw[end]) {
			end--
		}
		a.Raw = a.Raw[:end]
		a.Truncated = true
	}
	return a
}
