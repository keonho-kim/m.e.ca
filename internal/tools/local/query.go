// Purpose: Adapt parsed Plan listing/search commands to read-only query output.
// Pattern: Tool adapter; never starts a process or interprets shell code.
// Usage: operation.command delegates here for a prepared Plan query.
// Related: internal/policy/shell/query.go, internal/tools/local/search/search.go, internal/tools/local/directory.go.
package local

import (
	"context"
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	resultfmt "github.com/keonho-kim/v-code/internal/tools/result"
	"path/filepath"
	"strings"

	"github.com/keonho-kim/v-code/internal/tools/local/process"
	"github.com/keonho-kim/v-code/internal/tools/local/search"
)

func (op *operation) queryOutput(ctx context.Context) (process.Output, error) {
	q := *op.query
	output := process.Output{Status: "exited"}
	if q.Name != "ls" {
		result, err := (search.Request{Root: op.source.root, Path: op.source.relative, Private: op.executor.private, Query: q, Redact: op.executor.redact}).Execute(ctx)
		output.Stdout, output.Truncated = result.Text, result.Truncated
		if err != nil {
			output.ExitCode = 2
			output.Stderr = err.Error()
		} else if q.Content && result.Matches == 0 {
			output.ExitCode = 1
		}
		return output, err
	}
	info, err := op.source.root.Stat(op.source.relative)
	if err != nil {
		return process.Output{Status: "exited", ExitCode: 2, Stderr: err.Error()}, err
	}
	if !info.IsDir() {
		name := filepath.Base(q.Path)
		if q.Long {
			name = fmt.Sprintf("%s %10d %s", info.Mode(), info.Size(), name)
		}
		output.Stdout = name + "\n"
		return output, nil
	}
	entries, more, err := op.executor.Directory(ctx, op.settings, q.Path, 0, 1000)
	if err != nil {
		return process.Output{Status: "exited", ExitCode: 2, Stderr: err.Error()}, err
	}
	var text strings.Builder
	for _, entry := range entries {
		if !q.Hidden && strings.HasPrefix(entry.Name, ".") {
			continue
		}
		name := entry.Name
		if entry.Directory {
			name += "/"
		}
		if q.Long {
			info, err := op.source.root.Stat(filepath.Join(op.source.relative, entry.Name))
			if err != nil {
				return process.Output{Status: "exited", ExitCode: 2, Stderr: err.Error()}, err
			}
			name = fmt.Sprintf("%s %10d %s", info.Mode(), info.Size(), name)
		}
		if text.Len()+len(name)+1 > 64*1024 {
			more = true
			break
		}
		text.WriteString(name)
		text.WriteByte('\n')
	}
	output.Stdout, output.Truncated = text.String(), more
	return output, nil
}

func (op *operation) queryResult(ctx context.Context) (toolapi.Result, error) {
	output, err := op.queryOutput(ctx)
	output.Status = "query_completed"
	if err != nil {
		output.Status = "query_failed"
	}
	output.Stdout = op.executor.redact(output.Stdout)
	output.Stderr = op.executor.redact(output.Stderr)
	result, encodeErr := resultfmt.Encode(output, fmt.Sprintf("Plan read-only query; exit code %d", output.ExitCode))
	if encodeErr != nil {
		return result, encodeErr
	}
	result.Failed = err != nil || output.ExitCode != 0
	if err != nil {
		result.Failure = toolapi.FailureOf(err)
	}
	return result, err
}
