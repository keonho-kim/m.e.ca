// Purpose: Project conversational requirements and visible implementation choices for one current request.
// Pattern: Pure context projection.
// Usage: Main-agent decisions retain earlier requests and stated approaches.
// Related: internal/agent/context/request.go, internal/agent/run/work_context.go.
package context

import (
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"strings"
)

type Background struct {
	EarlierRequests  []string `json:"earlier_requests,omitempty"`
	EarlierWork      string   `json:"earlier_work,omitempty"`
	ProposedApproach []string `json:"proposed_approach,omitempty"`
}

// Work preserves the first stated approach and the latest action note. Both
// remain proposals; tool outcomes are the evidence of what actually happened.
func Work(current string, messages []modelapi.Message, memo string) Background {
	b := Background{EarlierWork: memo}
	// If compaction removed the request, the retained suffix belongs to its
	// ongoing work. Recent action notes still matter after that boundary.
	active := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" && messages[i].Content == current {
			active = i
			break
		}
	}
	seen := map[string]bool{}
	for i, m := range messages {
		if i < active && m.Role == "user" && !seen[m.Content] {
			b.EarlierRequests = append(b.EarlierRequests, m.Content)
			seen[m.Content] = true
		}
		if i <= active || m.Role != "assistant" || len(m.Calls) == 0 || strings.TrimSpace(m.Content) == "" {
			continue
		}
		if len(b.ProposedApproach) == 0 {
			b.ProposedApproach = append(b.ProposedApproach, m.Content)
			continue
		}
		if m.Content == b.ProposedApproach[0] {
			continue
		}
		if len(b.ProposedApproach) == 1 {
			b.ProposedApproach = append(b.ProposedApproach, m.Content)
		} else {
			b.ProposedApproach[1] = m.Content
		}
	}
	return b
}
