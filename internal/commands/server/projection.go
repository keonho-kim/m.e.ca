// Purpose: Project cumulative runtime snapshots into source-attributed deltas with explicit draft resets.
// Pattern: Bounded event projection.
// Usage: Service.emit handles runtime events; publish retains detached visible fields only.
// Related: internal/agent/run/generate.go, internal/commands/server/events.go.
package server

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
)

func prefix[T ~string | ~[]byte](value T, limit int) T {
	if len(value) <= limit {
		return value
	}
	for limit > 0 && !utf8.RuneStart(value[limit]) {
		limit--
	}
	return value[:limit]
}

func clipped(value string, limit int) string { return strings.Clone(prefix(value, limit)) }

// ProjectEvent bounds and detaches visible fields for replay and saved history.
func ProjectEvent(value Event) Event {
	if value.Outcome != nil {
		copy := *value.Outcome
		copy.Reason = clipped(copy.Reason, 4096)
		value.Outcome = &copy
	}
	value.MutationKey = clipped(value.MutationKey, 128)
	value.Status, value.Error = clipped(value.Status, 128), clipped(value.Error, 4096)
	e := value.Record
	if value.ToolPhase == "" {
		value.ToolPhase = e.ToolPhase()
	}
	p := record.Event{Kind: clipped(e.Kind, 128), Agent: clipped(e.Agent, 128), RunID: clipped(e.RunID, 128), Final: e.Final, Text: clipped(e.Text, maxTextBytes), Thinking: clipped(e.Thinking, maxTextBytes), CallID: clipped(e.CallID, 128), ApprovalSource: clipped(e.ApprovalSource, 128)}
	p.Kind, p.OccurredAt, p.ErrorCode = clipped(e.PublicKind(), 128), clipped(e.OccurredAt, 35), clipped(e.ErrorCode, 128)
	p.HistorySequence = e.HistorySequence
	p.MeasuredDuration = e.MeasuredDuration
	if e.Skill != nil {
		p.Skill = &record.SkillRead{ID: clipped(e.Skill.ID, 512), Name: clipped(e.Skill.Name, 64), Source: clipped(e.Skill.Source, 32), Revision: clipped(e.Skill.Revision, 64)}
	}
	value.Truncated = value.Truncated || len(e.Text) > maxTextBytes || len(e.Thinking) > maxTextBytes || len(e.Output) > 0 || len(e.ChangePreview) > 0
	if e.Reference != nil {
		copy := *e.Reference
		p.Reference = &copy
	}
	if e.Call != nil {
		p.Call = &tool.Call{ExecutionID: clipped(e.Call.ExecutionID, 128), BatchID: clipped(e.Call.BatchID, 128), ID: clipped(e.Call.ID, 128), Name: clipped(e.Call.Name, 128), Arguments: bytes.Clone(prefix(e.Call.Arguments, maxTextBytes))}
		value.Truncated = value.Truncated || len(e.Call.Arguments) > maxTextBytes
	}
	if e.Approval != nil {
		a := e.Approval
		p.Approval = &tool.Approval{ID: clipped(a.ID, 128), Name: clipped(a.Name, 256), Target: clipped(a.Target, 4096), Preview: clipped(a.Preview, maxTextBytes), Diff: a.Diff, Required: a.Required, UserRequired: a.UserRequired}
		value.Truncated = value.Truncated || len(a.Preview) > maxTextBytes
	}
	if e.Usage != nil {
		u := e.Usage.Clone()
		p.Usage = &u
	}
	if e.Statistics != nil {
		stats := *e.Statistics
		if stats.Usage != nil {
			u := stats.Usage.Clone()
			stats.Usage = &u
		}
		p.Statistics = &stats
	}
	if e.Generation != nil {
		g := *e.Generation
		g.ID = clipped(g.ID, 128)
		p.Generation = &g
	}
	if e.Team != nil {
		t := *e.Team
		value.Truncated = value.Truncated || len(t.Mentions) > 16
		t.Mentions = nil
		for _, mention := range e.Team.Mentions {
			if len(t.Mentions) == 16 {
				break
			}
			t.Mentions = append(t.Mentions, clipped(mention, 128))
			value.Truncated = value.Truncated || len(mention) > 128
		}
		if e.Team.Work != nil {
			w := *e.Team.Work
			value.Truncated = value.Truncated || len(w.Contract) > maxTextBytes || len(w.Paths) > 32
			w.Contract, w.Paths = clipped(w.Contract, maxTextBytes), nil
			for _, path := range e.Team.Work.Paths {
				if len(w.Paths) == 32 {
					break
				}
				w.Paths = append(w.Paths, clipped(path, 4096))
				value.Truncated = value.Truncated || len(path) > 4096
			}
			t.Work = &w
		}
		p.Team = &t
	}
	value.Record = p
	return value
}

func eventBytes(value Event) int {
	e := value.Record
	// Include every retained string; fixed overhead covers the value structs,
	// backing slices and small allocations rather than replacing text accounting.
	n := 1024 + len(value.Cursor) + len(value.Session) + len(value.Task) + len(value.Status) + len(value.Error) + len(value.ApprovalID)
	n += len(value.Workspace) + len(value.MutationKey)
	if value.Outcome != nil {
		n += len(value.Outcome.Status) + len(value.Outcome.Reason)
	}
	n += len(e.Kind) + len(e.Agent) + len(e.RunID) + len(e.Text) + len(e.Thinking) + len(e.CallID) + len(e.ApprovalSource)
	n += len(e.OccurredAt) + len(e.ErrorCode)
	if e.Skill != nil {
		n += len(e.Skill.ID) + len(e.Skill.Name) + len(e.Skill.Source) + len(e.Skill.Revision)
	}
	if e.Reference != nil {
		n += len(e.Reference.Session)
	}
	if e.Call != nil {
		n += len(e.Call.ExecutionID) + len(e.Call.ID) + len(e.Call.Name) + len(e.Call.Arguments)
	}
	if e.Approval != nil {
		n += len(e.Approval.ID) + len(e.Approval.Name) + len(e.Approval.Preview) + len(e.Approval.Target)
	}
	if e.Generation != nil {
		n += len(e.Generation.ID) + len(e.Generation.Role)
	}
	if e.Team != nil {
		n += len(e.Team.ReplyAuthor)
		for _, m := range e.Team.Mentions {
			n += len(m)
		}
		if e.Team.Work != nil {
			n += len(e.Team.Work.Contract)
			for _, p := range e.Team.Work.Paths {
				n += len(p)
			}
		}
	}
	return n
}
