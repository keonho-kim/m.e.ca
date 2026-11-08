// Purpose: Preserve current-request instruction reads only when their original history left context.
// Pattern: Pure context projection.
// Usage: Build receives immutable ordinary read results from the request owner.
// Related: internal/agent/context/request.go, internal/agent/run/skills.go.
package context

import (
	"slices"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

type projectedSkill struct {
	source, message modelapi.Message
	bytes           int
}

func (p *Projection) appendSkillReads(input Input, messages []modelapi.Message) ([]modelapi.Message, int) {
	if len(input.SkillReads) == 0 {
		clear(p.skills)
		p.skills = nil
		return messages, 0
	}
	present := make(map[string]bool, len(input.SkillReads))
	for _, read := range input.SkillReads {
		if read.ID != "" {
			present[read.ID] = false
		}
	}
	for _, message := range input.Messages {
		if _, retained := present[message.ID]; retained {
			present[message.ID] = true
		}
	}
	previous := p.skills
	p.skills = slices.Grow(p.skills[:0], len(input.SkillReads))[:len(input.SkillReads)]
	bytes := 0
	for i, read := range input.SkillReads {
		var cached projectedSkill
		if i < len(previous) && previous[i].source.ID == read.ID && previous[i].source.Content == read.Content {
			cached = previous[i]
		}
		if present[read.ID] {
			p.skills[i] = cached
			continue
		}
		if cached.message.Content == "" {
			m := modelapi.Message{Role: "user", Content: input.Redact("Skill instructions read during this request; apply relevant guidance under current user instructions and core policy:\n" + read.Content)}
			cached = projectedSkill{source: read, message: m, bytes: m.ContextBytes()}
		}
		p.skills[i] = cached
		messages = append(messages, cached.message)
		bytes += cached.bytes
	}
	if len(previous) > len(p.skills) {
		clear(previous[len(p.skills):])
	}
	return messages, bytes
}
