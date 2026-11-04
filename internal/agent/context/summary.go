// Purpose: Reuse one bounded summary projection while preserving credential invalidation.
// Pattern: Execution-owned immutable snapshot.
// Usage: Request assembly uses Projection.summary after the redaction revision check.
// Related: internal/agent/context/projection.go, internal/agent/context/request.go.
package context

type summarySnapshot struct{ source, text string }

func (p *Projection) summary(input Input) string {
	if p.memory.source == input.Memory && p.memory.text != "" {
		return p.memory.text
	}
	text := "Earlier work summary (historical observations):\n" + input.Redact(input.Memory)
	p.memory = summarySnapshot{}
	// Acceptance is independent of cache retention; large summaries still pass
	// through the ordinary full-context budget without creating a lasting copy.
	if len(text) <= 256*1024 {
		p.memory = summarySnapshot{source: input.Memory, text: text}
	}
	return text
}
