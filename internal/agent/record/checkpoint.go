// Purpose: Define durable execution status and validate resumable loop checkpoints.
// Pattern: State Machine contracts.
// Usage: Request runtime emits checkpoints and SQLite validates stored state.
// Related: internal/agent/run/journal.go, internal/storage/sqlite/runs.go.
package record

import (
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/agent/work"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type Status string

const (
	Running     Status = "running"
	Answered    Status = "answered"
	Stopped     Status = "stopped"
	Interrupted Status = diagcatalog.Interrupted
)

type Checkpoint struct {
	End          *work.End     `json:"end,omitempty"`
	Work         toolapi.Facts `json:"work"`
	ReportReason string        `json:"report_reason,omitempty"`
	LastChange   string        `json:"last_change,omitempty"`
	Blocker      string        `json:"blocker,omitempty"`

	ChangeRevision int `json:"change_revision"`

	ID                 string `json:"id"`
	Objective          string `json:"objective"`
	Status             Status `json:"status"`
	Phase              string `json:"phase"`
	Iteration          int    `json:"iteration"`
	ModelCalls         int    `json:"model_calls"`
	InstructionsDigest string `json:"instructions_digest,omitempty"`
	Tool               string `json:"tool,omitempty"`
	Observation        string `json:"observation,omitempty"`
}

func (c Checkpoint) Check() error {
	if c.End != nil {
		if err := c.End.Check(); err != nil {
			return err
		}
	}
	if err := c.Work.Check(); err != nil {
		return err
	}
	if len(c.ReportReason) > 256*1024*1024 {
		return faults.Errorf(diagcatalog.RecordInvalidReportReason)
	}
	if len(c.Blocker) > 16*1024*1024 || len(c.LastChange) > 16*1024*1024 {
		return faults.Errorf(diagcatalog.RecordInvalidBlockerDetail)
	}
	if c.ChangeRevision < 0 {
		return faults.Errorf(diagcatalog.RecordInvalidChangeRevision)
	}
	if c.ID == "" || len(c.ID) > 128 || len(c.Objective) > 64*1024*1024 || c.Iteration < 0 || c.ModelCalls < 0 || len(c.Observation) > 256*1024*1024 || (c.InstructionsDigest != "" && len(c.InstructionsDigest) != 64) {
		return faults.Errorf(diagcatalog.RecordInvalidExecutionCheckpoint)
	}
	switch c.Status {
	case Running, Answered, Stopped, Interrupted:
	default:
		return faults.Errorf(diagcatalog.RecordInvalidExecutionStatus)
	}
	switch c.Phase {
	case "decision", "approval", "execution", "observation", "reporting", "finished":
	default:
		return faults.Errorf(diagcatalog.RecordInvalidExecutionPhase)
	}
	return nil
}
