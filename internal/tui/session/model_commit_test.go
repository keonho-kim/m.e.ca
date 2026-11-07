// Purpose: Keep terminal input responsive while accepting a selection durably committed after cancellation.
// Pattern: Controlled I/O lifecycle regression test.
// Usage: Model-setting tests await worker completion without arbitrary sleeps.
// Related: internal/tui/session/model_commit.go, internal/tui/session/setup_cancel.go.
package session

import (
	"context"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
	"github.com/keonho-kim/meca/internal/provider/identity"
)

func finishModelChange(t *testing.T, a *app) {
	t.Helper()
	limit := time.NewTimer(3 * time.Second)
	defer limit.Stop()
	for a.work.state.active() {
		select {
		case n := <-a.events:
			a.receive(n)
		case <-limit.C:
			t.Fatal("model change did not finish")
		}
	}
}

type delayedSelection struct {
	Store
	started, proceed chan struct{}
}

func (s *delayedSelection) SaveSelection(ctx context.Context, id string, selected connection.Selection) error {
	close(s.started)
	<-s.proceed
	if err := s.Store.SaveSelection(context.WithoutCancel(ctx), id, selected); err != nil {
		return err
	}
	return ctx.Err()
}

func TestModelCommitCancellationAcceptsDurableResult(t *testing.T) {
	a := fixture(t)
	store := &delayedSelection{Store: a.opts.Store, started: make(chan struct{}), proceed: make(chan struct{})}
	a.opts.Store = store
	c, _ := connection.New(connection.Ollama, "")
	client, err := httpclient.NewClient(c, "")
	if err != nil {
		t.Fatal(err)
	}
	selected := connection.Selection{Connection: c, Model: "saved"}
	a.startModelCommit(notice{Client: client, Selection: &selected})
	<-store.started
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if !a.work.state.stopping() {
		t.Fatal("terminal input blocked during persistence")
	}
	close(store.proceed)
	finishModelChange(t, a)
	if a.connection.selection.Model != "saved" || a.connection.client != client {
		t.Fatal("durable result discarded after cancellation")
	}
	if a.stopNotice != "" {
		t.Fatal("adopted connection still displays a cancellation notice", a.stopNotice)
	}
}

func TestModelCommitDoesNotMutateBorrowedIdentity(t *testing.T) {
	a := fixture(t)
	if err := a.opts.Vault.Remember("secret-evidence"); err != nil {
		t.Fatal(err)
	}
	c, _ := connection.New(connection.Ollama, "")
	client, err := httpclient.NewClient(c, "")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	report := &identity.Report{Facts: []identity.Fact{{Source: "provider", Field: "label", Value: "secret-evidence"}}}
	next := connection.Selection{Connection: c, Model: "saved", Identity: report}
	n := commitPreparedModel(a.ctx, a.opts, notice{Client: client, Selection: &next})
	if n.Err != nil {
		t.Fatal(n.Err)
	}
	if len(report.Facts) != 1 || report.Incomplete {
		t.Fatal("worker mutated borrowed identity metadata", report)
	}
	if len(n.Commit.Selection.Identity.Facts) != 0 {
		t.Fatal("persisted identity retained a credential")
	}
}
