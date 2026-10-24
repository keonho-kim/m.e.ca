// Purpose: Reuse sanitized immutable history while invalidating changed messages and native groups.
// Pattern: Request-scoped projection cache.
// Usage: One execution owns Projection; Build bypasses it without a redaction revision source.
// Related: internal/agent/context/request.go, internal/agent/context/conversation.go, internal/agent/run/context.go.
package context

import (
	"bytes"
	"fmt"
	"slices"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type projectedMessage struct {
	source, message modelapi.Message
	paired          bool
	bytes           int
}

// Messages and their call/state bytes are immutable after entering execution
// history. This cache retains references, not another copy of large bodies.
type Projection struct {
	entries []projectedMessage
	version int
	model   string
}

func (p *Projection) project(input Input) ([]modelapi.Message, int, error) {
	for range 2 {
		messages, total, err := p.projectOnce(input)
		if err != nil {
			*p = Projection{}
			return nil, 0, err
		}
		if input.RedactionVersion() == p.version {
			return messages, total, nil
		}
		*p = Projection{}
	}
	return nil, 0, fmt.Errorf("protected credentials changed while assembling context; rebuild the request")
}

func (p *Projection) projectOnce(input Input) ([]modelapi.Message, int, error) {
	version := input.RedactionVersion()
	if p.version != version || p.model != input.Request.ModelID {
		*p = Projection{version: version, model: input.Request.ModelID}
	}
	paired := pairedTools(input.Messages)
	previous := p.entries
	p.entries = slices.Grow(p.entries[:0], len(input.Messages))[:len(input.Messages)]
	messages := make([]modelapi.Message, 0, len(input.Messages))
	total := 0
	for i, message := range input.Messages {
		var entry projectedMessage
		if i < len(previous) && previous[i].paired == paired[i] && sameMessage(previous[i].source, message) {
			entry = previous[i]
		} else {
			projected, err := conversationMessage(message, paired[i])
			if err != nil {
				return nil, 0, err
			}
			projected.Content = input.Redact(projected.Content)
			projected.Reasoning = input.Redact(projected.Reasoning)
			state := string(projected.ReasoningState)
			if input.Redact(state) != state {
				return nil, 0, fmt.Errorf("provider continuation state contains protected content")
			}
			entry = projectedMessage{source: message, message: projected, paired: paired[i], bytes: messageBytes(projected)}
		}
		p.entries[i] = entry
		messages = append(messages, entry.message)
		total += entry.bytes
	}
	if len(previous) > len(p.entries) {
		clear(previous[len(p.entries):])
	}
	return messages, total, nil
}

func sameMessage(a, b modelapi.Message) bool {
	return a.ID == b.ID && a.Role == b.Role && a.Content == b.Content && a.Reasoning == b.Reasoning && a.Incomplete == b.Incomplete && a.ToolCallID == b.ToolCallID && a.ToolName == b.ToolName && a.ProviderID == b.ProviderID && a.ModelID == b.ModelID && bytes.Equal(a.ReasoningState, b.ReasoningState) && slices.EqualFunc(a.Calls, b.Calls, func(a, b toolapi.Call) bool {
		return a.ID == b.ID && a.Name == b.Name && bytes.Equal(a.Arguments, b.Arguments)
	})
}

func messageBytes(m modelapi.Message) int {
	n := len(m.Content) + len(m.Reasoning) + len(m.ReasoningState) + len(m.ToolCallID) + len(m.ToolName)
	for _, call := range m.Calls {
		n += len(call.ID) + len(call.Name) + len(call.Arguments)
	}
	return n
}
