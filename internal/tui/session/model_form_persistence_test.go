// Purpose: Verify confirmed model settings persist credentials and controls while partial failures remain explicit.
// Pattern: Temporary-storage, mock-keyring and HTTP integration tests.
// Usage: go test ./internal/tui/session -run TestModelForm.
// Related: internal/tui/session/model_form_setup.go, internal/provider/setup/commit.go, internal/config/model.go.
package session

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
	"github.com/keonho-kim/meca/internal/provider/setup"
	modelsettings "github.com/keonho-kim/meca/internal/tui/settings/model"
	"github.com/zalando/go-keyring"
)

func editModelValue(a *app, row int, value string) {
	a.modelForm.Row = row
	a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlU, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, value, 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
}

func TestModelFormConfirmedGlobalSettingsSurviveReconnect(t *testing.T) {
	keyring.MockInit()
	a := fixture(t)
	secret := "private-form-test-key"
	var posts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			posts.Add(1)
			t.Error("model settings generated a response")
		}
		if r.URL.Path != "/api/v1/models" || r.Header.Get("Authorization") != "Bearer "+secret {
			t.Error("wrong metadata route or credential")
		}
		fmt.Fprint(w, `{"data":[{"id":"alias","reasoning":{"supported_efforts":["low","high"],"default_effort":"low","default_enabled":false}}]}`)
	}))
	defer server.Close()
	a.openModel()
	f := a.modelForm
	for i, p := range connection.Providers() {
		if p.Kind == connection.OpenRouter {
			f.Provider = i
		}
	}
	editModelValue(a, modelsettings.Endpoint, server.URL)
	editModelValue(a, modelsettings.Credential, secret)
	f.Row = modelsettings.Name
	a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	drainTask(t, a, false)
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "ali", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
	f.Row = modelsettings.Thinking
	a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	drainTask(t, a, false)
	if f.Thinking == nil || *f.Thinking || f.Effort != "low" {
		t.Fatal("declared OFF and strength were not resolved")
	}
	if a.client != nil || a.selection.Model != "" || a.opts.Config.Model != nil {
		t.Fatal("metadata preparation committed the draft")
	}
	if _, err := os.Stat(filepath.Join(a.opts.State, "credentials.used")); !os.IsNotExist(err) {
		t.Fatalf("key saved before confirmation: %v", err)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	f.Row = modelsettings.Effort
	a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "y", 0))
	if a.page != "chat" || a.modelForm != nil || a.selection.Effort() != "high" || a.selection.Thinking == nil || !*a.selection.Thinking {
		t.Fatalf("saved=%+v status=%s", a.selection, a.status)
	}
	cfg, err := config.Load(a.opts.State)
	if err != nil || cfg.Model == nil {
		t.Fatalf("config=%+v err=%v", cfg, err)
	}
	saved, err := cfg.Model.Selection()
	if err != nil || saved.Effort() != "high" || saved.Thinking == nil || !*saved.Thinking {
		t.Fatalf("stored defaults=%+v err=%v", saved, err)
	}
	key, err := a.opts.Vault.Get(saved.Connection.CredentialRef)
	if err != nil || key != secret {
		t.Fatalf("credential=%q err=%v", key, err)
	}
	body, err := os.ReadFile(filepath.Join(a.opts.State, config.Filename))
	if err != nil || strings.Contains(string(body), secret) {
		t.Fatal("credential leaked into config")
	}
	prepared, err := setup.Prepare(a.ctx, saved.Connection, "", saved.Model, setup.Defaults{}, saved, a.opts.Vault)
	if err != nil {
		t.Fatal(err)
	}
	defer prepared.Client.Close()
	if prepared.Selection.Effort() != "high" || prepared.Selection.Thinking == nil || !*prepared.Selection.Thinking || posts.Load() != 0 {
		t.Fatal("saved controls lost on reconnect or diagnostic was generated")
	}
}

func TestModelFormGlobalFailureKeepsCommittedSession(t *testing.T) {
	a := fixture(t)
	c, err := connection.New(connection.VLLM, "")
	if err != nil {
		t.Fatal(err)
	}
	a.client, err = httpclient.NewClient(c, "")
	if err != nil {
		t.Fatal(err)
	}
	off := false
	a.selection = connection.Selection{Connection: c, Model: "alias", Thinking: &off, ThinkingSupported: true}
	a.opts.SaveConfig = func(config.Config) error { return errors.New("default write failed") }
	a.openModel()
	a.modelForm.Row = modelsettings.Thinking
	a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "y", 0))
	if a.page != "chat" || a.modelForm != nil || !strings.Contains(a.status, "default write failed") || a.selection.Thinking == nil || !*a.selection.Thinking {
		t.Fatalf("selection=%+v status=%s", a.selection, a.status)
	}
	saved, err := a.opts.Store.Selection(a.ctx, a.opts.Session.ID)
	if err != nil || saved.Thinking == nil || !*saved.Thinking || a.opts.Config.Model != nil {
		t.Fatalf("partial commit lost: %+v %v", saved, err)
	}
}
