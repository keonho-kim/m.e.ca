// Purpose: Define accumulated conversation usage, evidence ledgers and atomic compaction storage contracts.
// Pattern: Consumer-owned persistence contracts.
// Usage: Used by the compaction controller, task runtime and SQLite adapter.
// Related: internal/agent/compact/controller.go, internal/storage/sqlite/memory.go.
package compact

import (
	"context"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent"
)

type Settings struct {
	ThresholdTokens int64 `toml:"threshold_tokens"`
	MaxSummaryBytes int   `toml:"max_summary_bytes"`
	MaxInputBytes   int   `toml:"max_input_bytes"`
}

func Defaults() Settings {
	return Settings{ThresholdTokens: 256000, MaxSummaryBytes: 16 * 1024, MaxInputBytes: 64 * 1024}
}
func (s Settings) Check() error {
	if s.ThresholdTokens < 1 || s.ThresholdTokens > 1000000000 || s.MaxSummaryBytes < 1024 || s.MaxSummaryBytes > 64*1024 || s.MaxInputBytes < 8192 || s.MaxInputBytes > 256*1024 || s.MaxSummaryBytes > s.MaxInputBytes/2 {
		return fmt.Errorf("invalid compaction settings")
	}
	return nil
}

type TokenCount struct {
	Amount    int64
	Estimated bool
	Kind      string
}

// Estimate uses a conservative byte upper bound, not a claim about a model tokenizer.
func Estimate(text string, kind string) TokenCount {
	return TokenCount{Amount: int64(len(text)), Estimated: true, Kind: kind}
}

type StepRef struct {
	PlanID          string `json:"plan_id"`
	Revision, Index int
	Description     string
	Objective       string
}
type TaskFact struct {
	ID, Objective, Status, Phase, Evidence string
	Observation                            string
	Step                                   *StepRef
	Source                                 int64
}
type Operation struct {
	TaskID, CallID, Name, Status, Command, Result string
	Check                                         bool
	Step                                          *StepRef
	Changes                                       []agent.FileChange
	Source                                        int64
}
type Record struct {
	Sequence   int64
	Role, Text string
	Task       *TaskFact
	Operation  *Operation
}
type Ledger struct {
	Tasks      []TaskFact
	Operations []Operation
}
type Memory struct {
	Revision              int
	Through, UsageThrough int64
	Markdown              string
	Ledger                Ledger
}
type Watermark struct {
	Tokens, Usage, Timeline int64
	Estimated               bool
}
type Repository interface {
	AddContextTokens(context.Context, string, TokenCount) error
	Memory(context.Context, string) (Memory, error)
	MemoryWatermark(context.Context, string, int64) (Watermark, error)
	MemoryRecords(context.Context, string, int64, int64) ([]Record, int64, error)
	InstallMemory(context.Context, string, int, Memory) error
}
type Generate func(context.Context, agent.ModelRequest, func(agent.ModelEvent) error) error
