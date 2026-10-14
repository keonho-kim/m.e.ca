// Purpose: Define accumulated conversation usage, evidence ledgers and atomic compaction storage contracts.
// Pattern: Consumer-owned persistence contracts.
// Usage: Used by the compaction controller, task runtime and SQLite adapter.
// Related: internal/agent/compact/controller.go, internal/storage/sqlite/memory.go.
package compact

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent/conclave"
	"github.com/keonho-kim/v-code/internal/agent/goal"
	"github.com/keonho-kim/v-code/internal/agent/grounding"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/plan"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/agent/verification"
)

type TaskFact struct {
	Checks                                 []verification.Record
	ChangeRevision                         int
	ID, Objective, Status, Phase, Evidence string
	Observation                            string
	Step                                   *plan.StepRef
	Source                                 int64
}
type GroundingFact struct {
	Record grounding.Record
	Source int64
}

type GoalFact struct {
	State  goal.State
	Source int64
}
type ReviewFact struct {
	Result conclave.Result
	Source int64
}

type Record struct {
	Goal       *goal.State
	Review     *conclave.Result
	Grounding  *grounding.Record
	Sequence   int64
	Role, Text string
	Task       *TaskFact
	Operation  *toolapi.Operation
}
type Ledger struct {
	Goals      []GoalFact
	Reviews    []ReviewFact
	Groundings []GroundingFact
	Tasks      []TaskFact
	Operations []toolapi.Operation
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
	CompactionBoundary(context.Context, string, int64, int) (int64, error)
	AddContextTokens(context.Context, string, TokenCount) error
	Memory(context.Context, string) (Memory, error)
	MemoryWatermark(context.Context, string, int64) (Watermark, error)
	MemoryRecords(context.Context, string, int64, int64) ([]Record, int64, error)
	InstallMemory(context.Context, string, int, Memory) error
}
type Generate func(context.Context, modelapi.Request, func(modelapi.Event) error) error
