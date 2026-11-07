// Purpose: Reuse sanitized immutable history while invalidating changed messages and native groups.
// Pattern: Request-scoped projection cache.
// Usage: One execution owns Projection; Build bypasses it without a redaction revision source.
// Related: internal/agent/context/request.go, internal/agent/context/conversation.go, internal/agent/run/context.go.
package context

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/redaction"
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
	system  instructionSnapshot
	memory  summarySnapshot
	work    summarySnapshot
}

func (p *Projection) build(input Input) (modelapi.Request, error) {
	for range 2 {
		version := input.RedactionVersion()
		if p.version != version || p.model != input.Request.ModelID {
			*p = Projection{version: version, model: input.Request.ModelID}
		}
		request, err := p.buildOnce(input)
		if err != nil {
			*p = Projection{}
			return modelapi.Request{}, err
		}
		if input.RedactionVersion() == p.version {
			return request, nil
		}
		*p = Projection{}
	}
	return modelapi.Request{}, fmt.Errorf("protected credentials changed while assembling context; rebuild the request")
}

func (p *Projection) appendHistory(input Input, messages []modelapi.Message) ([]modelapi.Message, int, error) {
	previous := p.entries
	common := 0
	for common < min(len(previous), len(input.Messages)) && sameMessage(previous[common].source, input.Messages[common]) {
		common++
	}
	from := pairingBoundary(input.Messages, common)
	p.entries = slices.Grow(p.entries[:0], len(input.Messages))[:len(input.Messages)]
	total, remaining := 0, 0
	for i, message := range input.Messages {
		paired := false
		if i < from {
			paired = previous[i].paired
		} else if remaining > 0 {
			paired = true
			remaining--
		} else if count := pairedGroup(input.Messages[i:]); count > 0 {
			paired = true
			remaining = count - 1
		}
		var entry projectedMessage
		if i < common && previous[i].paired == paired {
			entry = previous[i]
		} else {
			projected, err := conversationMessage(message, paired)
			if err != nil {
				return nil, 0, err
			}
			projected.Images = slices.Clone(projected.Images)
			for i := range projected.Images {
				projected.Images[i].Path = input.Redact(projected.Images[i].Path)
			}
			projected.Content = input.Redact(projected.Content)
			projected.Reasoning = input.Redact(projected.Reasoning)
			if err := redaction.CheckProtected(projected.ReasoningState, input.Redact); err != nil {
				return nil, 0, err
			}
			projected = projected.SealHistory()
			entry = projectedMessage{source: message, message: projected, paired: paired, bytes: projected.ContextBytes()}
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
	return samePublicSource(a, b) && sameImages(a.Images, b.Images) && a.ID == b.ID && a.Role == b.Role && a.Content == b.Content && a.Reasoning == b.Reasoning && a.Incomplete == b.Incomplete && a.ToolCallID == b.ToolCallID && a.ToolName == b.ToolName && a.ProviderID == b.ProviderID && a.ModelID == b.ModelID && bytes.Equal(a.ReasoningState, b.ReasoningState) && slices.EqualFunc(a.Calls, b.Calls, func(a, b toolapi.Call) bool {
		return a.ID == b.ID && a.Name == b.Name && bytes.Equal(a.Arguments, b.Arguments)
	})
}

// Images are immutable snapshots, so identity comparison does not scan their bytes.
func sameImages(a, b []media.Image) bool {
	return slices.EqualFunc(a, b, func(x, y media.Image) bool {
		return x.Path == y.Path && x.Digest == y.Digest && x.MIME == y.MIME && x.Format == y.Format && x.Representative == y.Representative && x.Width == y.Width && x.Height == y.Height && len(x.Data) == len(y.Data)
	})
}

func samePublicSource(a, b modelapi.Message) bool {
	if a.Source == nil || b.Source == nil {
		return a.Source == nil && b.Source == nil
	}
	return *a.Source == *b.Source
}
