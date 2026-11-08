// Purpose: Return a waiting candidate to its model when real public user steering arrives.
// Pattern: Temporary SQLite and controlled-executor integration test.
// Usage: Team admission tests keep a conflicting command target held while publishing a new request.
// Related: internal/tools/team/admission.go, internal/agent/team/decision.go.
package team

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"

	record "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/run"
	room "github.com/keonho-kim/meca/internal/agent/team"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"github.com/keonho-kim/meca/internal/tools/team/workspace"
)

type idleAdmission struct{ runningAdmission }

func (*idleAdmission) ActiveCommand() string { return "" }

func TestSteeringInterruptsWorkspacePreparationBeforeConflictingCommandEnds(t *testing.T) {
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
	held, err := w.ReserveCommand(ctx, []string{filepath.Join(root, "api.py")}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()
	ready, pending := make(chan struct{}, 1), make(chan string, 1)
	local := &idleAdmission{}
	var chosen, waited atomic.Bool
	result := make(chan error, 1)
	execute := func(ctx context.Context, _ identity.Member, _ string, p *room.Peer) (run.Result, error) {
		if !chosen.CompareAndSwap(false, true) {
			<-ctx.Done()
			return run.Result{}, ctx.Err()
		}
		p.Inbox(ctx)
		p.BeginDecision()
		_, err := p.SetWork("Create api.py", []string{filepath.Join(root, "api.py")}, "", 0)
		if err == nil {
			e := &Executor{Peer: p, Local: local, Workspace: w, Project: root}
			args, _ := json.Marshal(map[string]string{"path": "api.py", "content": "obsolete"})
			_, err = e.Prepare(ctx, policy.Settings{Scope: policy.System}, tool.Call{ID: "write", Name: "write_file", Arguments: args})
		}
		result <- err
		cancel()
		return run.Result{}, ctx.Err()
	}
	room.Run(ctx, room.Options{Store: store, Execute: execute, MaxModelCalls: 10, MaxSeconds: 5, ProjectRoot: root,
		SteeringReady: ready,
		PollSteering: func() []string {
			select {
			case text := <-pending:
				return []string{text}
			default:
				return nil
			}
		},
		Emit: func(event record.Event) error {
			if event.Kind == "tool_waiting" && waited.CompareAndSwap(false, true) {
				pending <- "Keep api.py unchanged"
				ready <- struct{}{}
			}
			return nil
		},
	}, "Implement API")
	err = <-result
	var failure *faults.Failure
	if !waited.Load() || local.prepared.Load() || !errors.As(err, &failure) || failure.Code != diagcatalog.InstructionChanged {
		t.Fatal("waiting operation did not return directly to a fresh decision", err)
	}
}
