// Purpose: Share current conversational background and select only execution facts absent from retained history.
// Pattern: Pure context projection.
// Usage: Main-agent decisions retain conversation background and compacted result gaps.
// Related: internal/agent/context/work.go, internal/agent/run/task_context.go.
package run

import (
	"github.com/keonho-kim/v-code/internal/agent/completion"
	contextbuild "github.com/keonho-kim/v-code/internal/agent/context"
	"slices"
)

func (s *execution) workBackground() contextbuild.Background {
	b := contextbuild.Work(s.checkpoint.Objective, s.messages, s.memoryContext)
	if previous := s.runner.PreviousRun; previous != nil && previous.Objective != "" && !slices.Contains(b.EarlierRequests, previous.Objective) {
		b.EarlierRequests = append(b.EarlierRequests, previous.Objective)
	}
	if redact := s.runner.Redact; redact != nil {
		b.EarlierWork = redact(b.EarlierWork)
		for i, text := range b.EarlierRequests {
			b.EarlierRequests[i] = redact(text)
		}
		for i, text := range b.ProposedApproach {
			b.ProposedApproach[i] = redact(text)
		}
	}
	return b
}

func (s *execution) missingResults(state completion.State) []taskResult {
	visible := map[string]bool{}
	for _, message := range s.messages {
		if message.Role == "tool" {
			visible[message.ToolCallID] = true
		}
	}
	missing := completion.State{}
	for _, receipt := range state.Receipts {
		if receipt.Stale {
			continue
		}
		id := receipt.ID
		if receipt.SourceCallID != "" {
			id = receipt.SourceCallID
		}
		if !visible[id] {
			missing.Receipts = append(missing.Receipts, receipt)
		}
	}
	return taskResults(missing)
}
