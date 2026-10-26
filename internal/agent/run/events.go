// Purpose: Define ordered execution events and the runtime persistence contract.
// Pattern: Consumer-owned event contracts.
// Usage: Runtime, SQLite and terminal projections exchange Event values.
// Related: internal/agent/run/journal.go, internal/storage/sqlite/events.go, internal/tui/transcript/events.go.
package run

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type Event struct {
	ApprovalSource  string               `json:"approval_source,omitempty"`
	Capture         json.RawMessage      `json:"capture,omitempty"`
	RunID           string               `json:"run_id,omitempty"`
	Generation      *Generation          `json:"generation,omitempty"`
	Statistics      *Statistics          `json:"statistics,omitempty"`
	ChangePreview   string               `json:"change_preview,omitempty"`
	TokensPerSecond *float64             `json:"-"`
	Internal        bool                 `json:"internal,omitempty"`
	Usage           *modelapi.TokenUsage `json:"usage,omitempty"`
	Call            *toolapi.Call        `json:"call,omitempty"`
	Output          json.RawMessage      `json:"output,omitempty"`
	Operation       *toolapi.Operation   `json:"operation,omitempty"`
	Kind            string               `json:"kind"`
	Text            string               `json:"text"`
	Thinking        string               `json:"thinking,omitempty"`
	CallID          string               `json:"call_id,omitempty"`
	Approval        *toolapi.Approval    `json:"approval,omitempty"`
	Checkpoint      *Checkpoint          `json:"checkpoint,omitempty"`
}

type Generation struct {
	ID            string         `json:"id"`
	Role          inference.Role `json:"role"`
	Attempt       int            `json:"attempt"`
	ElapsedMillis int64          `json:"elapsed_millis"`
}
type Journal interface {
	ContextMessages(context.Context, string, int64, int) ([]modelapi.Message, error)
	RecordEvent(context.Context, string, Event) error
	Commit(context.Context, string, []JournalRecord) error
}

// JournalRecord belongs to one durability boundary; exactly one payload must be set.
type JournalRecord struct {
	Message *modelapi.Message
	Event   *Event
}
