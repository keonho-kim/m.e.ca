// Purpose: Verify effort menus follow the serving API and failed persistence preserves the active preference.
// Pattern: Terminal settings behavior tests.
// Usage: go test ./internal/tui/session -run TestEffort.
// Related: internal/tui/session/effort.go, internal/tui/session/menus.go, internal/tui/view/header.go.
package session

import (
	"context"
	"github.com/gdamore/tcell/v3"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/httpclient"
	"reflect"
	"strings"
	"testing"
)

func TestEffortMenuAndModelFooter(t *testing.T) {
	a := fixture(t)
	connection, err := modelconn.New(modelconn.VLLM, "")
	if err != nil {
		t.Fatal(err)
	}
	medium := "medium"
	on := true
	a.selection = modelconn.Selection{Connection: connection, Model: "Qwen/Qwen3.8-27B", Thinking: &on, ThinkingSupported: true, ReasoningEffort: &medium}
	a.openModel()
	a.keyEvent(tcell.NewEventKey(tcell.KeyDown, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyDown, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.page != "effort" || a.choice != 3 || !reflect.DeepEqual(a.choices, []string{"DEFAULT", "MINIMAL", "LOW", "MEDIUM", "HIGH", "XHIGH", "MAX"}) {
		t.Fatalf("effort menu=%s %v index=%d", a.page, a.choices, a.choice)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	a.render()
	if got := strings.TrimSpace(screenLine(a.screen, 24-a.footerRows, 80)); got != a.selection.Model {
		t.Fatalf("unexpected model footer: %q", got)
	}
	off := false
	a.selection.Thinking = &off
	a.render()
	if got := strings.TrimSpace(screenLine(a.screen, 24-a.footerRows, 80)); got != a.selection.Model {
		t.Fatalf("OFF still displays strength: %q", got)
	}
	if a.selection.Effort() != "medium" {
		t.Fatal("turning off erased strength preference")
	}
}

func TestEffortUnavailableConnectionMenuCannotOpen(t *testing.T) {
	a := fixture(t)
	connection, err := modelconn.New(modelconn.VLLM, "")
	if err != nil {
		t.Fatal(err)
	}
	a.selection = modelconn.Selection{Connection: connection, Model: "alias", Details: &modelconn.ModelInfo{ID: "alias", Reasoning: &modelconn.Reasoning{On: true, Off: true}}}
	a.openModel()
	a.choice = 2
	a.choose()
	if a.page != "model" || a.status == "" {
		t.Fatal("undeclared effort menu opened")
	}
}

func TestEffortPersistenceFailurePreservesPreference(t *testing.T) {
	a := fixture(t)
	previous := "low"
	connection, err := modelconn.New(modelconn.VLLM, "")
	if err != nil {
		t.Fatal(err)
	}
	a.selection = modelconn.Selection{Model: "Qwen/Qwen3.8-27B", Connection: connection, ThinkingSupported: true}
	a.selection.ReasoningEffort = &previous
	if err := closeTestStore(t, a.opts.Store); err != nil {
		t.Fatal(err)
	}
	next := "medium"
	err = a.applyEffort(next)
	if a.selection.Effort() != "low" || err == nil {
		t.Fatal("save failure silently changed preference")
	}
}

func TestEffortSelectionPersistsWithoutGeneratingDiagnostic(t *testing.T) {
	a := fixture(t)
	conn, err := modelconn.New(modelconn.VLLM, "")
	if err != nil {
		t.Fatal(err)
	}
	a.client, err = httpclient.NewClient(conn, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.client.Close)
	low := "low"
	off := false
	a.selection = modelconn.Selection{Connection: conn, Model: "Qwen/Qwen3.8-27B", Thinking: &off, ThinkingSupported: true, ReasoningEffort: &low}
	a.openEffort()
	a.choice = 3
	a.chooseEffort()
	if a.busy || a.selection.Effort() != "medium" || *a.selection.Thinking {
		t.Fatalf("unexpected selection %+v", a.selection)
	}
	stored, err := a.opts.Store.Selection(context.Background(), a.opts.Session.ID)
	if err != nil || stored.Effort() != "medium" {
		t.Fatalf("%+v %v", stored, err)
	}
}
