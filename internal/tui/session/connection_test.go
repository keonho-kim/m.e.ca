// Purpose: Verify failed connection persistence preserves the active client and selection.
// Pattern: Session state transition tests.
// Usage: go test ./internal/tui/session -run TestConnectionSaveFailure.
// Related: internal/tui/session/events.go, internal/storage/sqlite/selection.go.
package session

import (
	"github.com/keonho-kim/v-code/internal/config"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/httpclient"
	"github.com/zalando/go-keyring"
	"testing"
)

func TestConnectionSaveFailure(t *testing.T) {
	a := fixture(t)
	conn, err := connection.New(connection.VLLM, "http://localhost:8000")
	if err != nil {
		t.Fatal(err)
	}
	old, err := httpclient.NewClient(conn, "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer old.Close()
	candidate, err := httpclient.NewClient(conn, "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer candidate.Close()
	a.client = old
	a.selection = connection.Selection{Connection: conn, Model: "old"}
	a.opts.Session.ID = "missing-session"
	next := connection.Selection{Connection: conn, Model: "new"}
	a.receive(notice{Done: true, Client: candidate, Selection: &next})
	if a.client != old || a.selection.Model != "old" || a.status == "" {
		t.Fatalf("failed save applied: model=%s status=%s", a.selection.Model, a.status)
	}
}

func TestConnectionCredentialDoesNotOverwriteSavedKey(t *testing.T) {
	keyring.MockInit()
	a := fixture(t)
	conn, err := connection.New(connection.VLLM, "http://localhost:8000")
	if err != nil {
		t.Fatal(err)
	}
	conn.CredentialRef = conn.ID
	if err := a.opts.Vault.Set(conn.CredentialRef, "old-key"); err != nil {
		t.Fatal(err)
	}
	candidate, err := httpclient.NewClient(conn, "new-key", "")
	if err != nil {
		t.Fatal(err)
	}
	defer candidate.Close()
	next := connection.Selection{Connection: conn, Model: "alias"}
	a.receive(notice{Done: true, Client: candidate, Selection: &next, credential: "new-key"})
	if a.status != "" {
		t.Fatal(a.status)
	}
	if a.selection.Connection.CredentialRef == conn.CredentialRef {
		t.Fatal("new key reused old reference")
	}
	old, err := a.opts.Vault.Get(conn.CredentialRef)
	if err != nil || old != "old-key" {
		t.Fatalf("saved key overwritten: %v", err)
	}
	saved, err := a.opts.Store.Selection(a.ctx, a.opts.Session.ID)
	if err != nil {
		t.Fatal(err)
	}
	current, err := a.opts.Vault.Get(saved.Connection.CredentialRef)
	if err != nil || current != "new-key" {
		t.Fatalf("new key not bound to persisted selection: %v", err)
	}
}

func TestThinkingSaveFailurePreservesChoice(t *testing.T) {
	for _, global := range []bool{false, true} {
		t.Run(map[bool]string{false: "session", true: "global"}[global], func(t *testing.T) {
			a := fixture(t)
			conn, err := connection.New(connection.VLLM, "http://localhost:8000")
			if err != nil {
				t.Fatal(err)
			}
			off := false
			a.selection = connection.Selection{Connection: conn, Model: "alias", Thinking: &off, ThinkingSupported: true}
			a.opts.Session.ID = "missing-session"
			a.changeValue = "ON"
			a.applySetting(global)
			if *a.selection.Thinking || a.status == "" {
				t.Fatalf("failed save applied: %+v", a.selection)
			}
			saved, err := config.Load(a.opts.State)
			if err != nil {
				t.Fatal(err)
			}
			if saved.Thinking || a.opts.Config.Thinking {
				t.Fatal("failed session save retained global change")
			}
		})
	}
}
