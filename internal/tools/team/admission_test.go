// Purpose: Reject work that would wait on the same peer's running command before acquiring leases.
// Pattern: Room and executor regression test with simulated command state.
// Usage: Team tests exercise actual admission without launching an OS process.
// Related: internal/tools/team/admission.go, internal/tools/team/workspace/workspace.go.
package team

import (
	"context"
	"encoding/json"
	"errors"
	record "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/run"
	room "github.com/keonho-kim/meca/internal/agent/team"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
	tool "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"github.com/keonho-kim/meca/internal/tools/team/workspace"
	"path/filepath"
	"sync/atomic"
	"testing"
)

type runningAdmission struct {
	Local
	prepared atomic.Bool
}

func (l *runningAdmission) ActiveCommand() string { return "cmd_active" }
func (l *runningAdmission) Prepare(context.Context, policy.Settings, tool.Call) (tool.Execution, error) {
	l.prepared.Store(true)
	return nil, errors.New("unexpected local preparation")
}

func TestOwnActiveCommandReturnsControlFeedbackBeforeWorkspaceWait(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := sqlite.OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := catalog.Create(context.Background(), root, root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := workspace.New()
	held, err := w.ReserveShell(ctx, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()
	var chosen, waited atomic.Bool
	local := &runningAdmission{}
	result := make(chan error, 1)
	execute := func(ctx context.Context, _ identity.Member, _ string, p *room.Peer) (run.Result, error) {
		if !chosen.CompareAndSwap(false, true) {
			<-ctx.Done()
			return run.Result{}, ctx.Err()
		}
		p.Inbox(ctx)
		p.BeginDecision()
		if _, err := p.SetWork("Create api.py", []string{filepath.Join(root, "api.py")}, "", 0); err != nil {
			result <- err
			cancel()
			return run.Result{}, err
		}
		e := &Executor{Peer: p, Local: local, Workspace: w, Project: root}
		args, _ := json.Marshal(map[string]string{"path": "api.py", "content": "x"})
		_, err := e.Prepare(ctx, policy.Settings{Scope: policy.System}, tool.Call{ID: "write", Name: "write_file", Arguments: args})
		result <- err
		cancel()
		return run.Result{}, ctx.Err()
	}
	room.Run(ctx, room.Options{Store: store, Execute: execute, MaxModelCalls: 10, MaxSeconds: 5, ProjectRoot: root, Emit: func(e record.Event) error {
		if e.Kind == "tool_waiting" {
			waited.Store(true)
			cancel()
		}
		return nil
	}}, "Implement api")
	err = <-result
	var failure *tool.Failure
	if waited.Load() || local.prepared.Load() || !errors.As(err, &failure) || failure.Code != "team_command_pending" {
		t.Fatal("file preparation did not return control before waiting", err)
	}
}
