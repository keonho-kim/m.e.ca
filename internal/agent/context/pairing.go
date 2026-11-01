// Purpose: Recheck only native groups that can be affected by an appended or changed suffix.
// Pattern: Allocation-free bounded group matching.
// Usage: Projection preserves completed prefix flags and checks groups from pairingBoundary.
// Related: internal/agent/context/projection.go, internal/agent/context/conversation.go.
package context

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func pairingBoundary(messages []modelapi.Message, common int) int {
	from := common
	for from > 0 && messages[from-1].Role == "tool" {
		from--
	}
	if from > 0 && messages[from-1].Role == "assistant" && len(messages[from-1].Calls) > 0 {
		from--
	}
	return from
}

// pairedGroup returns assistant plus result count, or zero for an orphan group.
func pairedGroup(messages []modelapi.Message) int {
	if len(messages) == 0 {
		return 0
	}
	first := messages[0]
	count := len(first.Calls)
	if first.Role != "assistant" || first.Incomplete || count == 0 || count > toolapi.MaxCalls || count >= len(messages) {
		return 0
	}
	var seen uint64
	for _, result := range messages[1 : count+1] {
		if result.Role != "tool" {
			return 0
		}
		matched := -1
		for index, call := range first.Calls {
			if call.ID == result.ToolCallID {
				if call.Name == result.ToolName {
					matched = index
				}
				break
			}
		}
		if matched < 0 || seen&(1<<matched) != 0 {
			return 0
		}
		seen |= 1 << matched
	}
	return count + 1
}
