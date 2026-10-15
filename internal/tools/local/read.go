// Purpose: Bind successful source reads to stable whole-file versions for later freshness checks.
// Pattern: Filesystem read adapter.
// Usage: Operation dispatch calls readFile after resolving the target and policy.
// Related: internal/tools/local/file/read.go, internal/tools/local/version.go.
package local

import (
	"context"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	fileops "github.com/keonho-kim/v-code/internal/tools/local/file"
)

func (op *operation) readFile(ctx context.Context) (toolapi.Result, error) {
	before, beforeErr := snapshot(op.source)
	result, err := (fileops.Read{Root: op.source.root, Path: op.source.relative, Absolute: op.source.absolute, Offset: op.input.Offset, Limit: op.input.Limit, Anchors: &op.executor.reads, Redact: op.executor.redact}).Execute(ctx)
	if err != nil {
		return result, err
	}
	after, afterErr := snapshot(op.source)
	if beforeErr == nil && afterErr == nil && sameVersion(before, after) {
		result.Versions = map[string]string{op.source.absolute: versionValue(after)}
	}
	return result, nil
}
