// Purpose: Select exactly one semantic rendering body for a public execution event.
// Pattern: Pure allowlisted transport projection.
// Usage: Event and history adapters share this boundary; control events carry no rendering body.
// Related: internal/serverapi/projection.go, api/meca/v1/events.proto.
package serverapi

import (
	"slices"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/commands/server"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func eventContent(result *v1.Event, value server.Event) {
	e := value.Record
	if completion := historyCompletion(e); completion != nil {
		result.Content = &v1.Event_Completion{Completion: completion}
		result.Statistics = nil
		return
	}
	switch result.Kind {
	case "assistant_delta", "thinking", "progress", "team_progress", "team_message", "team_discussion", "user_message":
		m := &v1.TranscriptMessage{Role: "assistant", Text: e.Text}
		if result.Kind == "thinking" || result.Kind == "assistant_delta" {
			m.Thinking = e.Thinking
		}
		if e.Agent == "user" || result.Kind == "user_message" {
			m.Role = "user"
		}
		if result.Kind == "team_progress" || result.Kind == "team_message" || result.Kind == "team_discussion" {
			m.Team = wireTeam(e.Team)
		}
		result.Content = &v1.Event_Message{Message: m}
	case "team_work":
		result.Content = &v1.Event_Work{Work: &v1.TeamContribution{Summary: e.Text, Details: wireTeam(e.Team)}}
	case "tool_started", "tool_waiting", "tool_succeeded", "tool_failed", "tool_interrupted", "tool_skipped", "command_finished", "denied", "skill_loaded":
		if e.Call != nil || e.CallID != "" {
			phase := value.ToolPhase
			if phase == "" {
				phase = e.ToolPhase()
			}
			t := &v1.Tool{CallId: e.CallID, Phase: wireToolPhase(phase), Summary: e.Text}
			t.Skill = wireSkillRead(e.Skill)
			if e.Call != nil {
				t.ExecutionId, t.BatchId, t.CallId, t.Name, t.ArgumentsJson = e.Call.ExecutionID, e.Call.BatchID, e.Call.ID, e.Call.Name, e.Call.Arguments
			}
			if e.Reference != nil {
				t.Reference = &v1.EventReference{SessionId: e.Reference.Session, Sequence: e.Reference.Sequence}
			}
			result.Content = &v1.Event_Tool{Tool: t}
		} else if result.Kind == "skill_loaded" && e.Skill != nil {
			result.Content = &v1.Event_Notice{Notice: &v1.Notice{Code: v1.Notice_SKILL_READ, Skill: wireSkillRead(e.Skill)}}
			if e.Reference != nil {
				result.OriginalReference = &v1.EventReference{SessionId: e.Reference.Session, Sequence: e.Reference.Sequence}
			}
		}
	case "retry":
		result.Content = &v1.Event_Notice{Notice: &v1.Notice{Code: v1.Notice_RETRYING}}
	case "compaction_started":
		result.Content = &v1.Event_Notice{Notice: &v1.Notice{Code: v1.Notice_COMPACTION_STARTED}}
	case "compaction_finished":
		result.Content = &v1.Event_Notice{Notice: &v1.Notice{Code: v1.Notice_COMPACTION_FINISHED}}
	}
}

func wireSkillRead(s *record.SkillRead) *v1.SkillRead {
	if s == nil {
		return nil
	}
	return &v1.SkillRead{Id: s.ID, Name: s.Name, Source: s.Source, Revision: s.Revision}
}

func wireTeam(t *record.TeamNote) *v1.Team {
	if t == nil {
		return nil
	}
	value := &v1.Team{ReplyTo: t.ReplyTo, ReplyAuthor: t.ReplyAuthor, Mentions: slices.Clone(t.Mentions), BasedOn: t.BasedOn}
	if t.Work != nil {
		value.Paths, value.Contract, value.Revision = slices.Clone(t.Work.Paths), t.Work.Contract, t.Work.Revision
	}
	return value
}
