// Purpose: Define ordered execution events and the runtime persistence contract.
// Pattern: Consumer-owned event contracts.
// Usage: Runtime, SQLite and terminal projections exchange Event values.
// Related: internal/agent/run/journal.go, internal/storage/sqlite/events.go, internal/tui/transcript/events.go.
package run

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/agent/conclave"
	"github.com/keonho-kim/v-code/internal/agent/grounding"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/monitor"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

type Event struct {
	Internal   bool                 `json:"internal,omitempty"`
	Monitor    *monitor.Record      `json:"monitor,omitempty"`
	Conclave   *conclave.Result     `json:"conclave,omitempty"`
	Grounding  *grounding.Record    `json:"grounding,omitempty"`
	Usage      *modelapi.TokenUsage `json:"usage,omitempty"`
	Call       *toolapi.Call        `json:"call,omitempty"`
	Output     json.RawMessage      `json:"output,omitempty"`
	Operation  *toolapi.Operation   `json:"operation,omitempty"`
	Kind       string               `json:"kind"`
	Text       string               `json:"text"`
	Thinking   string               `json:"thinking,omitempty"`
	CallID     string               `json:"call_id,omitempty"`
	Approval   *toolapi.Approval    `json:"approval,omitempty"`
	Checkpoint *Checkpoint          `json:"checkpoint,omitempty"`
}
type Journal interface {
	ContextMessages(context.Context, string, int64, int) ([]modelapi.Message, error)
	AppendMessage(context.Context, string, modelapi.Message) error
	RecordEvent(context.Context, string, Event) error
}
