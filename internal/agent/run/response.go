// Purpose: Stream ordinary replies while withholding unverified completion claims.
// Pattern: Presentation projection.
// Usage: Model generation calls responseBody for unfinished output.
// Related: internal/agent/run/finish.go, internal/agent/run/generate.go.
package run

func (s *execution) responseBody(text string) string {
	if s.runner.suppressStream || s.checkpoint.PendingReview || s.changed && s.checkpoint.Evidence == "" {
		return ""
	}
	return text
}
