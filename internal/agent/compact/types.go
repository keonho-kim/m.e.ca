// Purpose: Define conversation source records, summary memory and atomic compaction storage contracts.
// Pattern: Consumer-owned persistence contracts.
// Usage: Used by the compaction controller, task runtime and SQLite adapter.
// Related: internal/agent/compact/controller.go, internal/storage/sqlite/memory.go.
package compact

import (
	"context"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

type Record struct {
	Sequence   int64
	Role, Text string
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
