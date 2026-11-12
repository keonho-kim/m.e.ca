// Purpose: Convert visible domain values into the versioned wire schema and mask secret strings.
// Pattern: Pure external projection.
// Usage: Command replies and all event transports call these conversions.
// Related: api/meca/v1/state.proto, api/meca/v1/events.proto.
package serverapi

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/redaction"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"github.com/keonho-kim/meca/internal/storage/history"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func task(value server.Task) *v1.Task {
	return &v1.Task{Id: value.ID, SessionId: value.Session, Status: value.Status, Error: value.Error, Code: value.Code, Persistence: value.Persistence, Outcome: value.Outcome, SessionSummary: value.SessionSummary}
}

func snapshot(value server.Snapshot) *v1.Snapshot {
	value = server.BoundSnapshot(value)
	s := &v1.Snapshot{WorkspaceId: value.Workspace, Cursor: value.Cursor, Permission: value.Permission, Team: value.Team, Session: wireSession(value.Session), Model: wireModel(value.Selection), Concurrency: int64(value.Concurrency), NextTask: value.NextTask, TotalTasks: int64(value.TotalTasks), Language: value.Language, ActivityTruncated: value.ActivityTruncated}
	for _, t := range value.Tasks {
		s.Tasks = append(s.Tasks, task(t))
	}
	for _, x := range value.Executions {
		s.Executions = append(s.Executions, &v1.Execution{TaskId: x.Task, Session: wireSession(x.Session), Model: wireModel(x.Selection), Team: x.Team, Statistics: wireStatistics(x.Statistics), StartedAt: wireTime(x.StartedAt)})
	}
	if value.Task != nil {
		s.Task = task(*value.Task)
	}
	if x := value.Execution; x != nil {
		s.Execution = &v1.Execution{TaskId: x.Task, Session: wireSession(x.Session), Model: wireModel(x.Selection), Team: x.Team, Statistics: wireStatistics(x.Statistics), StartedAt: wireTime(x.StartedAt)}
	}
	for _, p := range value.Approvals {
		s.Approvals = append(s.Approvals, event(server.Event{Task: p.Task, ApprovalID: p.ID, ApprovalExpiresAt: p.ExpiresAt, Record: p.Value, Truncated: p.Truncated}).Approval)
	}
	for _, d := range value.Drafts {
		s.Drafts = append(s.Drafts, &v1.Draft{TaskId: d.Task, Agent: d.Agent, Generation: d.Generation, Text: d.Text, Thinking: d.Thinking, Truncated: d.Truncated, TextBytes: d.TextBytes, ThinkingBytes: d.ThinkingBytes, Committed: d.Committed})
	}
	for _, e := range value.Activity {
		s.Activity = append(s.Activity, event(e))
	}
	return s
}

func wireSession(value sessionstate.Session) *v1.Session {
	if value.ID == "" {
		return nil
	}
	return &v1.Session{Id: value.ID, ProjectRoot: value.ProjectRoot, DisplayRoot: value.DisplayRoot}
}

func wireModel(m connection.Selection) *v1.Model {
	if m.Model == "" {
		return nil
	}
	reasoning := connection.ReasoningOptions(m.Connection.Kind, m.Details)
	value := &v1.Model{ReasoningLevels: reasoning.Levels, ReasoningOff: reasoning.Off, Provider: string(m.Connection.Kind), Endpoint: m.Connection.Endpoint, Model: m.Model, CredentialSaved: m.Connection.CredentialRef != "", Thinking: m.Thinking, ThinkingSupported: m.ThinkingSupported}
	if m.ReasoningEffort != nil {
		value.ReasoningEffort = new(*m.ReasoningEffort)
	}
	if m.Details != nil {
		value.ContextWindow, value.Notice = int32(m.Details.ContextWindow), m.Details.Notice
		value.ImageInputState, value.ImageInputSupported = m.Details.ImageInput.State, m.Details.ImageInput.Supported()
	}
	return value
}

func event(value server.Event) *v1.Event {
	e := value.Record
	result := &v1.Event{MutationKey: value.MutationKey, WorkspaceId: value.Workspace, Cursor: value.Cursor, SessionId: value.Session, TaskId: value.Task, Agent: e.Agent, RunId: e.RunID, Generation: value.Generation, Kind: e.Kind, Text: e.Text, Thinking: e.Thinking, Reset_: value.Reset, Truncated: value.Truncated, Final: e.Final, ApprovalId: value.ApprovalID, ApprovalSource: e.ApprovalSource}
	if value.Outcome != nil {
		result.Outcome = &v1.RunOutcome{Status: string(value.Outcome.Status), Reason: value.Outcome.Reason}
	}
	result.Status, result.Error = value.Status, value.Error
	result.Sequence = value.Sequence
	result.MeasuredMillis = e.MeasuredDuration.Milliseconds()
	result.OccurredAt, result.ErrorCode = e.OccurredAt, e.ErrorCode
	if e.Skill != nil {
		result.Skill = &v1.SkillRead{Id: e.Skill.ID, Name: e.Skill.Name, Source: e.Skill.Source, Revision: e.Skill.Revision}
	}
	if a := e.Approval; a != nil {
		result.Approval = &v1.Approval{ExpiresAt: wireTime(value.ApprovalExpiresAt), TaskId: value.Task, Id: value.ApprovalID, CallId: a.ID, Name: a.Name, Target: a.Target, Preview: a.Preview, Diff: a.Diff, UserRequired: a.UserRequired, Truncated: value.Truncated}
	}
	if e.Call != nil || e.Reference != nil || e.CallID != "" {
		phase := value.ToolPhase
		if phase == "" {
			phase = e.ToolPhase()
		}
		result.Tool = &v1.Tool{CallId: e.CallID, Phase: wireToolPhase(phase)}
		if e.Call != nil {
			result.Tool.ExecutionId = e.Call.ExecutionID
			result.Tool.BatchId = e.Call.BatchID
			result.Tool.CallId = e.Call.ID
			result.Tool.Name = e.Call.Name
			result.Tool.ArgumentsJson = e.Call.Arguments
		}
		if e.Reference != nil {
			result.Tool.Reference = &v1.EventReference{SessionId: e.Reference.Session, Sequence: e.Reference.Sequence}
		}
	}
	if e.Usage != nil {
		result.Usage = &v1.Usage{Input: e.Usage.Input, Output: e.Usage.Output, Cost: e.Usage.Cost}
	}
	if e.Statistics != nil {
		stats := e.Statistics
		result.Statistics = &v1.Statistics{Concurrent: stats.Concurrent, ElapsedMillis: stats.ElapsedMillis, GenerationMillis: stats.GenerationMillis}
		if stats.Usage != nil {
			u := stats.Usage
			result.Statistics.Usage = &v1.Usage{Input: u.Input, Output: u.Output, Cost: u.Cost}
		}
	}
	if e.Generation != nil {
		g := e.Generation
		result.Measurement = &v1.GenerationMeasurement{Id: g.ID, Role: string(g.Role), Attempt: int32(g.Attempt), ElapsedMillis: g.ElapsedMillis, MeasuredMillis: g.MeasuredMillis}
	}
	if t := e.Team; t != nil {
		// Sanitization mutates the outgoing message. Each subscriber owns its
		// list backing arrays; retained domain events stay immutable.
		result.Team = &v1.Team{ReplyTo: t.ReplyTo, ReplyAuthor: t.ReplyAuthor, Mentions: slices.Clone(t.Mentions), BasedOn: t.BasedOn}
		if t.Work != nil {
			result.Team.Paths = slices.Clone(t.Work.Paths)
			result.Team.Contract = t.Work.Contract
			result.Team.Revision = t.Work.Revision
		}
	}
	return result
}

func textPreview(text string) string {
	const limit = 16 << 10
	if len(text) <= limit {
		return text
	}
	n := limit
	for n > 0 && !utf8.RuneStart(text[n]) {
		n--
	}
	return text[:n]
}

func historyItem(value history.Record) *v1.HistoryItem {
	item := &v1.HistoryItem{Sequence: value.Sequence, Kind: value.Kind}
	if completion := historyCompletion(value.Event); value.Kind == "event" && completion != nil {
		item.Kind, item.Completion = "completion", completion
		return item
	}
	for index, image := range value.Images {
		item.Images = append(item.Images, wireImage(index, image))
	}
	if value.Kind == "message" {
		item.Role = value.Message.Role
		item.Text = strings.Clone(textPreview(value.Message.Content))
		item.Truncated = value.Truncated || len(item.Text) < len(value.Message.Content)
		item.Incomplete = value.Message.Incomplete
	} else {
		item.Event = event(server.ProjectEvent(server.Event{Record: value.Event}))
	}
	return item
}

func (a *API) sanitize(value proto.Message) {
	if a.Redact == nil {
		return
	}
	var visit func(protoreflect.Message)
	visit = func(m protoreflect.Message) {
		clipped := false
		if field := m.Descriptor().Fields().ByName("truncated"); field != nil {
			clipped = m.Get(field).Bool()
		}
		m.Range(func(f protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			if f.IsList() {
				list := v.List()
				for i := 0; i < list.Len(); i++ {
					if f.Kind() == protoreflect.MessageKind {
						visit(list.Get(i).Message())
					} else if f.Kind() == protoreflect.StringKind {
						list.Set(i, protoreflect.ValueOfString(a.Redact(list.Get(i).String())))
					}
				}
			} else if f.Kind() == protoreflect.MessageKind {
				visit(v.Message())
			} else if f.Kind() == protoreflect.StringKind {
				redact := a.Redact
				if clipped && a.RedactPreview != nil && (f.Name() == "text" || f.Name() == "thinking" || f.Name() == "preview" || f.Name() == "diff") {
					redact = a.RedactPreview
				}
				m.Set(f, protoreflect.ValueOfString(redact(v.String())))
			} else if f.Kind() == protoreflect.BytesKind && f.Name() == "arguments_json" {
				body, err := redaction.RedactJSON(v.Bytes(), a.Redact)
				if err != nil {
					m.Clear(f)
				} else {
					m.Set(f, protoreflect.ValueOfBytes(body))
				}
			}
			return true
		})
	}
	visit(value.ProtoReflect())
}

func notificationCursor(n *v1.Notification) string {
	if s := n.GetSnapshot(); s != nil {
		return s.Cursor
	}
	if e := n.GetEvent(); e != nil {
		return e.Cursor
	}
	return ""
}

func validCorrelation(id string) bool {
	return len(id) > 0 && len(id) <= 128 && !strings.ContainsAny(id, "\r\n\x00")
}
