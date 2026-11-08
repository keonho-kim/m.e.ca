// Purpose: Read explicit user skill mentions and retain ordinary read results for this request's context.
// Pattern: Request input and result handling.
// Usage: Initial input, steering and committed tool outcomes use the same reader and journal.
// Related: internal/skills/instruction/document.go, internal/agent/run/action_result.go, internal/agent/context/skill_reads.go.
package run

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func skillReadMetadata(doc instruction.Document) *runrecord.SkillRead {
	return &runrecord.SkillRead{ID: doc.ID, Name: doc.Name, Source: doc.Source, Revision: doc.Revision}
}

func (s *execution) readMentions(ctx context.Context, text string) error {
	names := instruction.Mentions(text)
	if len(names) == 0 {
		return nil
	}
	for _, name := range names {
		var doc instruction.Document
		found := false
		for _, preloaded := range s.runner.SkillReads {
			if preloaded.Name == name || preloaded.ID == name {
				doc, found = preloaded, true
				break
			}
		}
		if !found {
			if s.runner.Skills == nil {
				return faults.Errorf(diagcatalog.RunSkillReaderIsUnavailableForExplicitMentions)
			}
			var err error
			doc, err = s.runner.Skills.Load(ctx, instruction.Selector{ID: name}, true)
			if err != nil {
				return faults.Errorf(diagcatalog.RunReadExplicitlyRequestedSkill, name, err)
			}
		}
		s.allowExplicitSkill(doc.ID)
		body, err := json.Marshal(doc)
		if err != nil {
			return err
		}
		if err := s.saveSkillRead(ctx, doc, string(body)); err != nil {
			return err
		}
	}
	return nil
}

func (s *execution) saveSkillRead(ctx context.Context, doc instruction.Document, body string) error {
	if version := s.skillReadRevisions["skill:"+doc.ID]; version == doc.Revision {
		return nil
	}
	content := "Explicitly requested skill instructions:\n" + body
	message := modelapi.Message{ID: "skill_" + rand.Text(), Role: "system", Content: content}
	message.Content = s.runner.Redact(message.Content)
	if err := s.checkInstruction("skill:"+doc.ID, "", message); err != nil {
		return err
	}
	event := runrecord.Event{Kind: "skill_loaded", Skill: skillReadMetadata(doc), Text: "Read skill " + doc.Name}
	event.OccurredAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := s.save(ctx, message, event); err != nil {
		return err
	}
	if err := s.retainSkillRead(doc, s.messages[len(s.messages)-1]); err != nil {
		return err
	}
	if err := s.emit(event); err != nil {
		return err
	}
	return nil
}
