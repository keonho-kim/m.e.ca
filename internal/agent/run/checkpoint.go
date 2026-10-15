// Purpose: Define durable execution status and validate resumable loop checkpoints.
// Pattern: State Machine contracts.
// Usage: Task runtime emits checkpoints and SQLite validates stored state.
// Related: internal/agent/run/journal.go, internal/storage/sqlite/runs.go.
package run

import (
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/completion"
	"github.com/keonho-kim/v-code/internal/agent/conclave"
	"github.com/keonho-kim/v-code/internal/agent/grounding"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

type Status string

const (
	Running     Status = "running"
	Verified    Status = "verified"
	Answered    Status = "answered"
	Stopped     Status = "stopped"
	Unverified  Status = "unverified"
	Interrupted Status = "interrupted"
)

type Checkpoint struct {
	Work          completion.State `json:"work"`
	ReportReason  string           `json:"report_reason,omitempty"`
	LastChange    string           `json:"last_change,omitempty"`
	Blocker       string           `json:"blocker,omitempty"`
	PendingReview bool             `json:"pending_review"`
	Conclave      *conclave.Result `json:"conclave,omitempty"`

	ChangeRevision int               `json:"change_revision"`
	Grounding      *grounding.Record `json:"grounding,omitempty"`

	ID                 string                     `json:"id"`
	Objective          string                     `json:"objective"`
	Status             Status                     `json:"status"`
	Phase              string                     `json:"phase"`
	Iteration          int                        `json:"iteration"`
	ModelCalls         int                        `json:"model_calls"`
	NoProgress         int                        `json:"no_progress"`
	InstructionsDigest string                     `json:"instructions_digest"`
	Tool               string                     `json:"tool,omitempty"`
	Arguments          map[string]json.RawMessage `json:"arguments,omitempty"`
	Observation        string                     `json:"observation,omitempty"`
}

func (c Checkpoint) Check() error {
	if err := checkCompletion(c); err != nil {
		return err
	}
	if c.Conclave != nil {
		if err := c.Conclave.Check(); err != nil {
			return err
		}
	}
	if len(c.Blocker) > 16*1024*1024 || len(c.LastChange) > 16*1024*1024 {
		return fmt.Errorf("invalid blocker detail")
	}
	if c.ChangeRevision < 0 {
		return fmt.Errorf("invalid change revision")
	}
	if c.ChangeRevision < 0 || c.NoProgress < 0 {
		return fmt.Errorf("invalid progress state")
	}
	if c.Grounding != nil {
		if err := c.Grounding.Check(); err != nil {
			return err
		}
	}
	if c.ID == "" || len(c.ID) > 128 || len(c.Objective) > 16*1024*1024 || c.Iteration < 0 || c.ModelCalls < 0 || len(c.Observation) > 256*1024*1024 || len(c.InstructionsDigest) != 64 {
		return fmt.Errorf("invalid execution checkpoint")
	}
	switch c.Status {
	case Running, Verified, Answered, Stopped, Interrupted, Unverified:
	default:
		return fmt.Errorf("invalid execution status")
	}
	switch c.Phase {
	case "conclave", "grounding", "decision", "approval", "execution", "observation", "reporting", "completion_review", "finished":
	default:
		return fmt.Errorf("invalid execution phase")
	}
	body, err := json.Marshal(c.Arguments)
	if err != nil || len(body) > toolapi.MaxArgumentsBytes {
		return fmt.Errorf("invalid checkpoint arguments")
	}
	return nil
}
