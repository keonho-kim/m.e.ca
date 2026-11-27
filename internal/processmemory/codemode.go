// Purpose: Bound aggregate reservations for embedded JavaScript heaps.
// Pattern: Process-owned admission with one release per code execution.
// Usage: The code runtime reserves a heap before creating its VM.
// Related: internal/processmemory/transient.go, internal/tools/codemode/engine.go.
package processmemory

import "context"

const CodeBytes int64 = 256 << 20

var codeHeaps = transientBudget{maximum: CodeBytes}

func AcquireCode(ctx context.Context, bytes int64) (func(), error) {
	return codeHeaps.acquire(ctx, bytes)
}
