// Purpose: Fork a private LAT executor that exposes only query and owned command tools.
// Pattern: Capability-restricting executor adapter with owned cleanup.
// Usage: LAT runners call Executor.ReadOnly and defer the returned cleanup function.
// Related: internal/tools/local/read_only_command.go, internal/tools/local/executor.go.
package local

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/policy/readonly"
	toolcatalog "github.com/keonho-kim/meca/internal/tools/local/catalog"
	"github.com/keonho-kim/meca/internal/tools/local/textformat"
)

// The embedded executor is a newly owned instance, never the parent's registry,
// file handles or captures. The adapter replaces both tool entry points.
type readOnlyExecutor struct {
	*Executor
	mu       sync.Mutex
	running  sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	closed   atomic.Bool
	once     sync.Once
	closeErr error
	release  map[string]func()
}

// ReadOnly returns an isolated executor for LAT read/search work. It constrains
// the accepted query language; trusted host executables are not an OS sandbox.
func (parent *Executor) ReadOnly() (toolapi.Executor, func() error, error) {
	if err := parent.boundary.Check(); err != nil {
		return nil, nil, err
	}
	project, err := os.OpenRoot(parent.project)
	if err != nil {
		return nil, nil, err
	}
	system, err := os.OpenRoot(parent.systemRoot.Name())
	if err != nil {
		return nil, nil, errors.Join(err, project.Close())
	}
	child := &Executor{imageReads: make(chan struct{}, 1), project: parent.project, home: parent.home, projectRoot: project, systemRoot: system, redact: parent.redact, boundary: parent.boundary, commandWait: parent.commandWait}
	child.artifacts, child.artifactLease, err = newArtifacts()
	if err != nil {
		return nil, nil, errors.Join(err, child.Close())
	}
	child.commands.Save, child.commands.CaptureDirectory = child.archiveCommand, child.artifacts
	ctx, cancel := context.WithCancel(context.Background())
	adapter := &readOnlyExecutor{Executor: child, ctx: ctx, cancel: cancel}
	return adapter, adapter.close, nil
}

func (*readOnlyExecutor) Definitions() []toolapi.Definition {
	out := toolcatalog.Definitions("execute_command", "read_image", "list_command_runs", "wait_command", "cancel_command")
	for i := range out {
		definition := &out[i]
		if definition.Name == "execute_command" {
			definition.Description = "Run a LAT read-only query in project_root. Supports literal ls, cat, head, tail, wc, grep, rg, find, sed print ranges, uniq, cut, tr, du, stat, readlink, basename, dirname, printf, pwd and Git query subcommands. Pipes, && and sequential queries are allowed. No writes, redirections, variables, substitutions, unquoted globs, wrappers, interpreters, network, builds, tests or package commands. Unsupported options are rejected with guidance. Running results use this executor's own list_command_runs, wait_command and cancel_command."
			definition.Parameters[0].Description = "Literal POSIX read/search query. Quote patterns and paths; use find or rg instead of unquoted shell globs."
		}
	}
	return out
}

func readOnlyTool(name string) bool {
	return name == "execute_command" || name == "read_image" || commandControl(name)
}

func (executor *readOnlyExecutor) Prepare(ctx context.Context, settings policy.Settings, call toolapi.Call) (toolapi.Execution, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if executor.closed.Load() {
		return nil, fmt.Errorf("LAT read-only executor is closed")
	}
	if !readOnlyTool(call.Name) {
		return nil, fmt.Errorf("tool %q is unavailable in LAT read-only execution; use execute_command for read/search queries", call.Name)
	}
	if err := settings.Check(); err != nil {
		return nil, err
	}
	if call.Name != "execute_command" {
		prepared, err := executor.Executor.Prepare(ctx, settings, call)
		if err != nil {
			return nil, err
		}
		return &readOnlyOperation{executor: executor, delegated: prepared}, nil
	}
	spec, input, err := toolcatalog.Parse(call)
	if err != nil {
		return nil, err
	}
	if settings.Scope != policy.System {
		return nil, fmt.Errorf("LAT host queries require system Scope; directory confinement is unavailable")
	}
	if textformat.RedactSource(input.Command, executor.redact) != input.Command {
		return nil, fmt.Errorf("tool arguments contain a protected credential")
	}
	if _, err := readonly.Parse(input.Command); err != nil {
		return nil, err
	}
	return &readOnlyOperation{executor: executor, command: &operation{executor: executor.Executor, settings: settings, spec: spec, input: input, callID: call.ID}}, nil
}

type readOnlyOperation struct {
	executor  *readOnlyExecutor
	delegated toolapi.Execution
	command   *operation
	executed  atomic.Bool
}

func (op *readOnlyOperation) Approval() toolapi.Approval {
	if op.delegated != nil {
		return op.delegated.Approval()
	}
	return toolapi.Approval{Name: "Reading with LAT query", Target: op.command.input.Command}
}

func (op *readOnlyOperation) Execute(ctx context.Context, approved bool) (toolapi.Result, error) {
	if err := op.executor.enter(); err != nil {
		return toolapi.Result{}, err
	}
	defer op.executor.running.Done()
	ctx, cancel := context.WithCancel(ctx)
	stopped := make(chan struct{})
	stop := context.AfterFunc(op.executor.ctx, func() { cancel(); close(stopped) })
	release := func() {
		if !stop() {
			<-stopped
		}
		cancel()
	}
	defer func() {
		if release != nil {
			release()
		}
	}()
	if err := ctx.Err(); err != nil {
		return toolapi.Result{}, err
	}
	if !op.executed.CompareAndSwap(false, true) {
		return toolapi.Result{}, fmt.Errorf("prepared operation has already executed")
	}
	if op.delegated != nil {
		return op.delegated.Execute(ctx, approved)
	}
	result, err := op.executeQuery(ctx)
	if result.Process != nil && result.Process.Status == "running" {
		// A standalone caller need not bind a request registry first. Retain its
		// cancellation linkage until this executor's owned cleanup joins the job.
		op.executor.mu.Lock()
		if !op.executor.closed.Load() {
			if op.executor.release == nil {
				op.executor.release = make(map[string]func())
			}
			op.executor.release[result.Process.CommandID] = release
			release = nil
		}
		op.executor.mu.Unlock()
	}
	return result, err
}
