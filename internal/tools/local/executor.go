// Purpose: Prepare policy-checked tools and bind execution to immutable approved targets.
// Pattern: Adapter with a one-shot operation lifecycle.
// Usage: Injected into the agent; file tests invoke Prepare and Execute.
// Related: internal/agent/tools.go, internal/tools/local/paths.go, internal/tools/local/files.go, internal/tools/local/diff/diff.go.

package local

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/keonho-kim/v-code/internal/agent"
	"github.com/keonho-kim/v-code/internal/policy"
	"github.com/keonho-kim/v-code/internal/tools/local/diff"
)

type Executor struct {
	project, home, private  string
	projectRoot, systemRoot *os.Root
	redact                  func(string) string
}

func NewExecutor(project, home, private string, redact func(string) string) (*Executor, error) {
	if redact == nil {
		return nil, fmt.Errorf("a secret redactor is required")
	}
	for _, path := range []string{project, home, private} {
		if !filepath.IsAbs(path) {
			return nil, fmt.Errorf("filesystem roots must be absolute")
		}
	}
	project, err := realPath(project)
	if err != nil {
		return nil, err
	}
	home, err = realPath(home)
	if err != nil {
		return nil, err
	}
	private, err = realPath(private)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(project)
	if err != nil {
		return nil, err
	}
	system, err := os.OpenRoot(filepath.VolumeName(project) + string(filepath.Separator))
	if err != nil {
		return nil, errors.Join(err, root.Close())
	}
	return &Executor{project: project, home: home, private: private, projectRoot: root, systemRoot: system, redact: redact}, nil
}

func (executor *Executor) Close() error {
	return errors.Join(executor.projectRoot.Close(), executor.systemRoot.Close())
}

type operation struct {
	executor                     *Executor
	settings                     policy.Settings
	planning                     bool
	spec                         specification
	input                        arguments
	source, destination          target
	original, destinationVersion version
	replacement                  []byte
	approval                     agent.ApprovalRequest
	executed                     atomic.Bool
}

func (executor *Executor) Prepare(ctx context.Context, settings policy.Settings, planning bool, call agent.ToolCall) (agent.ToolExecution, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	spec, input, err := parseArguments(call)
	if err != nil {
		return nil, err
	}
	required, err := policy.Decide(settings, planning, spec.effect)
	if err != nil {
		return nil, err
	}
	for _, text := range []string{input.Content, input.OldText, input.NewText, input.Command, input.Pattern} {
		if executor.redact(text) != text {
			return nil, fmt.Errorf("tool arguments contain a protected credential")
		}
	}
	op := &operation{executor: executor, settings: settings, planning: planning, spec: spec, input: input}
	op.approval = agent.ApprovalRequest{ID: "operation_" + rand.Text(), Name: spec.label, Required: required}
	if spec.effect == policy.Command {
		op.approval.Target = input.Command
		op.approval.Preview = input.Command
		return op, nil
	}
	op.source, err = executor.resolve(settings.Scope, input.Path)
	if err != nil {
		return nil, err
	}
	op.approval.Target = op.source.absolute
	if spec.effect == policy.Read {
		if _, err := op.source.root.Stat(op.source.relative); err != nil {
			return nil, err
		}
		return op, nil
	}
	op.original, err = snapshot(op.source)
	if err != nil {
		return nil, err
	}
	if spec.name != "write_file" && !op.original.exists {
		return nil, fmt.Errorf("Path not found")
	}
	switch spec.name {
	case "write_file", "edit_file":
		if op.original.exists && op.original.info.IsDir() {
			return nil, fmt.Errorf("target is a directory")
		}
		var before []byte
		if op.original.exists {
			before, err = op.source.root.ReadFile(op.source.relative)
			if err != nil {
				return nil, err
			}
		}
		op.replacement = []byte(input.Content)
		if spec.name == "edit_file" {
			if input.OldText == "" || strings.Count(string(before), input.OldText) != 1 {
				return nil, fmt.Errorf("old_text must match exactly once")
			}
			op.replacement = []byte(strings.Replace(string(before), input.OldText, input.NewText, 1))
		}
		op.approval.Preview = executor.redact(diff.Preview(op.source.absolute, before, op.replacement, op.original.exists, true))
		op.approval.Diff = true
	case "move_file":
		op.destination, err = executor.resolve(settings.Scope, input.Destination)
		if err != nil {
			return nil, err
		}
		op.destinationVersion, err = snapshot(op.destination)
		if err != nil {
			return nil, err
		}
		if op.destinationVersion.exists {
			return nil, fmt.Errorf("destination already exists")
		}
		op.approval.Preview = "Move to " + op.destination.absolute
	case "delete_file":
		op.approval.Preview = "Delete " + op.source.absolute
		if !op.original.info.IsDir() {
			before, err := op.source.root.ReadFile(op.source.relative)
			if err != nil {
				return nil, err
			}
			op.approval.Preview = executor.redact(diff.Preview(op.source.absolute, before, nil, true, false))
			op.approval.Diff = true
		}
	}
	return op, nil
}

func (op *operation) Approval() agent.ApprovalRequest { return op.approval }

func (op *operation) Execute(ctx context.Context, approved bool) (agent.ToolResult, error) {
	if err := ctx.Err(); err != nil {
		return agent.ToolResult{}, err
	}
	required, err := policy.Decide(op.settings, op.planning, op.spec.effect)
	if err != nil {
		return agent.ToolResult{}, err
	}
	if required && !approved {
		return agent.ToolResult{}, fmt.Errorf("operation denied: approval required")
	}
	if !op.executed.CompareAndSwap(false, true) {
		return agent.ToolResult{}, fmt.Errorf("prepared operation has already executed")
	}
	if op.spec.effect != policy.Command {
		current, err := op.executor.resolve(op.settings.Scope, op.input.Path)
		if err != nil {
			return agent.ToolResult{}, err
		}
		if current.absolute != op.source.absolute {
			return agent.ToolResult{}, fmt.Errorf("target changed after preparation")
		}
		if op.spec.effect == policy.Change {
			version, err := snapshot(current)
			if err != nil {
				return agent.ToolResult{}, err
			}
			if !sameVersion(op.original, version) {
				return agent.ToolResult{}, fmt.Errorf("file changed after approval; prepare a new change")
			}
		}
	}
	return op.run(ctx)
}

func encodeResult(value any, summary string) (agent.ToolResult, error) {
	content, err := json.Marshal(value)
	return agent.ToolResult{Content: content, Summary: summary}, err
}
