// Purpose: Reuse sanitized current-work context without accumulating snapshots in conversation history.
// Pattern: Execution-owned bounded immutable projection.
// Usage: Build invalidates this view with the same credential revision barrier as other context.
// Related: internal/agent/context/projection.go, internal/agent/team/work_context.go.
package context

func (p *Projection) observations(input Input) string {
	if p.work.text != "" && p.work.source == input.Observations {
		return p.work.text
	}
	text := "Current team work (observed facts):\n" + input.Redact(input.Observations)
	p.work = summarySnapshot{}
	if len(text) <= 256*1024 {
		p.work = summarySnapshot{source: input.Observations, text: text}
	}
	return text
}
