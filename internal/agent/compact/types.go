// Purpose: Define conversation source records, summary memory and atomic compaction storage contracts.
// Pattern: Consumer-owned persistence contracts.
// Usage: Used by the compaction controller, task runtime and SQLite adapter.
// Related: internal/agent/compact/controller.go, internal/storage/sqlite/memory.go.
package compact

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent/conclave"
	"github.com/keonho-kim/v-code/internal/agent/grounding"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/plan"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

type RunFact struct {
	ChangeRevision               int
	ID, Objective, Status, Phase string
	Observation                  string
	Step                         *plan.StepRef
	Source                       int64
}
type Record struct {
	Review     *conclave.Result
	Grounding  *grounding.Record
	Sequence   int64
	Role, Text string
	Run        *RunFact
	Operation  *toolapi.Operation
}
type Memory struct {
	Revision              int
	Through, UsageThrough int64
	Markdown              string
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
