// Purpose: Verify two team peers overlap sixteen real commands each and keep native results ordered and durable.
// Pattern: Temporary SQLite integration test with a deterministic execution barrier.
// Usage: go test -race ./internal/tools/team -run TestTeamCommands.
// Related: internal/tools/team/parallel.go, internal/tools/team/parallel_fixture_test.go, internal/agent/run/batch_wave.go.
package team

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/run"
	agentsession "github.com/keonho-kim/meca/internal/agent/session"
	room "github.com/keonho-kim/meca/internal/agent/team"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/commands/agentrequest"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"github.com/keonho-kim/meca/internal/tools/local"
	"github.com/keonho-kim/meca/internal/tools/team/approval"
	"github.com/keonho-kim/meca/internal/tools/team/workspace"
)

func TestTeamCommandsOverlapAcrossPeersAndNativeBatch(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	dbs, err := sqlite.OpenSessions(filepath.Join(home, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	public, err := dbs.Create(context.Background(), root, root)
	if err != nil {
		t.Fatal(err)
	}
	defer public.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	access := workspace.New()
	entered, release, reports := make(chan string, 2*commandsPerPeer), make(chan struct{}), make(chan struct{})
	results := make(chan error, 2)
	expected := make([]string, commandsPerPeer)
	for i := range expected {
		expected[i] = fmt.Sprint(i)
	}
	execute := func(ctx context.Context, member identity.Member, _ string, peer *room.Peer) (run.Result, error) {
		child, err := dbs.Create(ctx, root, root)
		if err != nil {
			return run.Result{}, err
		}
		defer child.Close()
		native, err := local.NewExecutor(root, home, filepath.Join(home, ".meca"), func(s string) string { return s })
		if err != nil {
			return run.Result{}, err
		}
		defer native.Close()
		if _, err := peer.Inbox(ctx); err != nil {
			return run.Result{}, err
		}
		peer.BeginDecision()
		if _, err := peer.SetWork("Inspect independent command output", nil, "", 0); err != nil {
			return run.Result{}, err
		}
		tools := &Executor{Local: &barrierCommands{Executor: native, entered: entered, release: release, member: member.ID}, Peer: peer, Workspace: access, Refusals: &approval.Ledger{}, Project: root}
		model := &commandBatchModel{}
		action := agentrequest.New(agentrequest.Options{Runtime: agentrequest.Runtime{Config: config.Defaults(), Tools: tools,
			Session: child.Session(), Store: child, Vault: &credentials.Vault{State: filepath.Join(home, ".meca")},
			Selection:   connection.Selection{Model: "fixture"},
			LoadPrompts: func() (prompt.Templates, error) { return prompt.DefaultsFor(prompt.Team), nil }}, Text: "Inspect sixteen independent commands", Generator: model})
		var ordered []string
		result, err := agentsession.Execute(ctx, action, func(context.Context, toolapi.Approval) (bool, error) { return true, nil }, func(event runrecord.Event) error {
			if event.Kind == "tool_succeeded" {
				ordered = append(ordered, event.CallID)
			}
			return nil
		})
		if err == nil {
			messages, readErr := child.ContextMessages(ctx, child.Session().ID, 0, 16<<20, false)
			err = readErr
			var saved []string
			for _, message := range messages {
				if message.Role == "tool" {
					saved = append(saved, message.ToolCallID)
				}
			}
			if err == nil && !slices.Equal(saved, expected) {
				err = fmt.Errorf("%s: private SQLite command results=%v", member.ID, saved)
			}
		}
		if err == nil && (result.HadToolFailure || model.calls != 2 || !slices.Equal(ordered, expected)) {
			err = fmt.Errorf("%s: result=%+v model calls=%d ordered=%v", member.ID, result, model.calls, ordered)
		}
		results <- err
		select {
		case <-reports:
		case <-ctx.Done():
		}
		return result, err
	}
	done := make(chan error, 1)
	go func() {
		_, err := room.Run(ctx, room.Options{Store: public, Execute: execute, ProjectRoot: root, MaxSeconds: 10, MaxModelCalls: 20,
			Emit: func(runrecord.Event) error { return nil }}, "Run independent inspections")
		done <- err
	}()
	defer func() { cancel(); <-done }()
	seen := make(map[string]bool, 2*commandsPerPeer)
	for range 2 * commandsPerPeer {
		select {
		case id := <-entered:
			seen[id] = true
		case err := <-results:
			t.Fatalf("peer returned before parallel command entry: %v", err)
		case err := <-done:
			done <- err
			t.Fatalf("team stopped before command entry: %v", err)
		case <-ctx.Done():
			t.Fatalf("commands serialized before all peers could run: entered=%v", seen)
		}
	}
	if len(seen) != 2*commandsPerPeer {
		t.Fatalf("parallel entries repeated instead of covering every command: %v", seen)
	}
	close(release)
	for range 2 {
		select {
		case err := <-results:
			if err != nil {
				t.Fatal(err)
			}
		case <-ctx.Done():
			t.Fatal("parallel command results did not settle")
		}
	}
	close(reports)
	t.Logf("two peers, %d concurrent command entries, %d ordered results per private SQLite session", 2*commandsPerPeer, commandsPerPeer)
}
