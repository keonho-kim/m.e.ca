// Purpose: Project complete visible event content while keeping journal and execution-private fields off the wire.
// Pattern: Explicit public detail DTO mapping.
// Usage: document.go uses publicEvent for tool and saved-history originals without preview truncation.
// Related: internal/serverapi/detail/operation.go, internal/agent/record/events.go.
package detail

import (
	"encoding/json"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
)

type event struct {
	Kind           string          `json:"kind"`
	Agent          string          `json:"agent,omitempty"`
	Final          bool            `json:"final,omitempty"`
	RunID          string          `json:"runId,omitempty"`
	Text           string          `json:"text,omitempty"`
	Thinking       string          `json:"thinking,omitempty"`
	CallID         string          `json:"callId,omitempty"`
	ApprovalSource string          `json:"approvalSource,omitempty"`
	ChangePreview  string          `json:"changePreview,omitempty"`
	Call           *call           `json:"call,omitempty"`
	Output         json.RawMessage `json:"output,omitempty"`
	Operation      *operation      `json:"operation,omitempty"`
	Reference      *reference      `json:"reference,omitempty"`
	Usage          *usage          `json:"usage,omitempty"`
	Statistics     *statistics     `json:"statistics,omitempty"`
	Team           *team           `json:"team,omitempty"`
}

type call struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}
type reference struct {
	SessionID string `json:"sessionId"`
	Sequence  int64  `json:"sequence,string"`
}
type usage struct {
	Input  int64    `json:"input,string"`
	Output int64    `json:"output,string"`
	Cost   *float64 `json:"cost,omitempty"`
}
type statistics struct {
	Concurrent       bool   `json:"concurrent"`
	ElapsedMillis    int64  `json:"elapsedMillis,string"`
	GenerationMillis int64  `json:"generationMillis,string"`
	Usage            *usage `json:"usage,omitempty"`
}
type team struct {
	ReplyTo     int64    `json:"replyTo,string,omitempty"`
	ReplyAuthor string   `json:"replyAuthor,omitempty"`
	Mentions    []string `json:"mentions,omitempty"`
	BasedOn     int64    `json:"basedOn,string,omitempty"`
	Paths       []string `json:"paths,omitempty"`
	Contract    string   `json:"contract,omitempty"`
	Revision    uint64   `json:"revision,string,omitempty"`
}

func publicUsage(value *model.TokenUsage) *usage {
	if value == nil {
		return nil
	}
	return &usage{Input: value.Input, Output: value.Output, Cost: value.Cost}
}

func publicEvent(value record.Event) *event {
	e := &event{Kind: value.Kind, Agent: value.Agent, Final: value.Final, RunID: value.RunID, Text: value.Text, Thinking: value.Thinking, CallID: value.CallID, ApprovalSource: value.ApprovalSource, ChangePreview: value.ChangePreview, Output: value.Output, Usage: publicUsage(value.Usage)}
	if value.Call != nil {
		e.Call = &call{value.Call.ID, value.Call.Name, value.Call.Arguments}
	}
	if value.Reference != nil {
		e.Reference = &reference{value.Reference.Session, value.Reference.Sequence}
	}
	if value.Operation != nil {
		e.Operation = publicOperation(*value.Operation)
	}
	if s := value.Statistics; s != nil {
		e.Statistics = &statistics{s.Concurrent, s.ElapsedMillis, s.GenerationMillis, publicUsage(s.Usage)}
	}
	if t := value.Team; t != nil {
		e.Team = &team{ReplyTo: t.ReplyTo, ReplyAuthor: t.ReplyAuthor, Mentions: t.Mentions, BasedOn: t.BasedOn}
		if t.Work != nil {
			e.Team.Paths, e.Team.Contract, e.Team.Revision = t.Work.Paths, t.Work.Contract, t.Work.Revision
		}
	}
	return e
}
