// Purpose: Define task status, bounded execution budgets and durable loop checkpoints.
// Pattern: Consumer-owned task contracts.
// Usage: Shared by task orchestration, TOML settings and SQLite checkpoint recovery.
// Related: internal/agent/run/run.go, internal/storage/sqlite/tasks.go, internal/config/config.go.
package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent"
	"github.com/keonho-kim/v-code/internal/agent/compact"
	"github.com/keonho-kim/v-code/internal/policy"
)

type Status string

const (
	Running     Status = "running"
	Verified    Status = "verified"
	Answered    Status = "answered"
	NeedsInput  Status = "needs_input"
	Stopped     Status = "stopped"
	Interrupted Status = "interrupted"
	PlanReady   Status = "plan_ready"
)

type Limits struct {
	MaxIterations    int `toml:"max_iterations"`
	MaxModelCalls    int `toml:"max_model_calls"`
	MaxSeconds       int `toml:"max_seconds"`
	MaxModelSeconds  int `toml:"max_model_seconds"`
	MaxContextBytes  int `toml:"max_context_bytes"`
	MaxArgumentBytes int `toml:"max_argument_bytes"`
	StallIterations  int `toml:"stall_iterations"`
}

func DefaultLimits() Limits {
	return Limits{MaxIterations: 128, MaxModelCalls: 1024, MaxSeconds: 14400, MaxModelSeconds: 600, MaxContextBytes: 96 * 1024, MaxArgumentBytes: 64 * 1024, StallIterations: 3}
}
func (l Limits) Check() error {
	if l.MaxIterations < 1 || l.MaxIterations > 4096 || l.MaxModelCalls < 1 || l.MaxModelCalls > 16384 || l.MaxSeconds < 1 || l.MaxSeconds > 86400 || l.MaxModelSeconds < 1 || l.MaxModelSeconds > 3600 || l.MaxContextBytes < 8192 || l.MaxContextBytes > 256*1024 || l.MaxArgumentBytes < 1024 || l.MaxArgumentBytes > agent.MaxArgumentsBytes || l.StallIterations < 1 || l.StallIterations > 10 {
		return fmt.Errorf("invalid Ralph loop limits")
	}
	return nil
}

type Checkpoint struct {
	Step               *compact.StepRef           `json:"step,omitempty"`
	Checking           bool                       `json:"checking,omitempty"`
	ID                 string                     `json:"id"`
	Objective          string                     `json:"objective"`
	Status             Status                     `json:"status"`
	Phase              string                     `json:"phase"`
	Iteration          int                        `json:"iteration"`
	ModelCalls         int                        `json:"model_calls"`
	NoProgress         int                        `json:"no_progress"`
	InstructionsDigest string                     `json:"instructions_digest"`
	Tool               string                     `json:"tool,omitempty"`
	Parameter          string                     `json:"parameter,omitempty"`
	Arguments          map[string]json.RawMessage `json:"arguments,omitempty"`
	Observation        string                     `json:"observation,omitempty"`
	Evidence           string                     `json:"evidence,omitempty"`
}

func (c Checkpoint) Check() error {
	if c.ID == "" || len(c.ID) > 128 || len(c.Objective) > 64*1024 || c.Iteration < 0 || c.ModelCalls < 0 || len(c.Observation) > 16*1024 || len(c.Evidence) > 4096 || len(c.InstructionsDigest) != 64 {
		return fmt.Errorf("invalid task checkpoint")
	}
	switch c.Status {
	case Running, Verified, Answered, NeedsInput, Stopped, Interrupted, PlanReady:
	default:
		return fmt.Errorf("invalid task status")
	}
	switch c.Phase {
	case "decision", "argument", "approval", "execution", "observation", "finished":
	default:
		return fmt.Errorf("invalid task phase")
	}
	body, err := json.Marshal(c.Arguments)
	if err != nil || len(body) > agent.MaxArgumentsBytes {
		return fmt.Errorf("invalid checkpoint arguments")
	}
	return nil
}

type Event struct {
	Operation  *compact.Operation     `json:"operation,omitempty"`
	Kind       string                 `json:"kind"`
	Text       string                 `json:"text"`
	Thinking   string                 `json:"thinking,omitempty"`
	CallID     string                 `json:"call_id,omitempty"`
	Approval   *agent.ApprovalRequest `json:"approval,omitempty"`
	Checkpoint *Checkpoint            `json:"checkpoint,omitempty"`
}
type Journal interface {
	AppendMessage(context.Context, string, agent.Message) error
	RecordEvent(context.Context, string, Event) error
}

// Usage is owned by one serial worker and shared across its approved Plan steps.
type Usage struct{ Iterations, ModelCalls int }
type Runner struct {
	Generator                agent.Generator
	Tools                    agent.ToolExecutor
	Journal                  Journal
	Session, Model, Provider string
	ProjectInstructions      string
	BasePrompt               string
	Memory                   *compact.Controller
	Step                     *compact.StepRef
	Thinking                 *bool
	Settings                 policy.Settings
	Limits                   Limits
	Redact                   func(string) string
	MaskStream               func(string) string
	Usage                    *Usage
}
type Result struct {
	Messages       []agent.Message
	Text           string
	Status         Status
	Evidence       string
	HadToolFailure bool
}
type Approver func(context.Context, agent.ApprovalRequest) (bool, error)

var errApprovalDenied = errors.New("operation denied by user; task stopped")
