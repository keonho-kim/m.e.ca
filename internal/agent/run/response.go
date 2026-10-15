// Purpose: Stream model replies independently of verification status.
// Pattern: Presentation projection.
// Usage: Model generation calls responseBody for unfinished output.
// Related: internal/agent/run/finish.go, internal/agent/run/generate.go.
package run

func (s *execution) responseBody(text string) string {
	if s.runner.suppressStream {
		return ""
	}
	return text
}
