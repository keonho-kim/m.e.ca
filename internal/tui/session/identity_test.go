// Purpose: Verify model identity labels and evidence through terminal menu workflows.
// Pattern: Terminal workflow tests.
// Usage: go test ./internal/tui/session -run TestIdentity.
// Related: internal/tui/settings/model/identity.go, internal/tui/session/render_model.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/provider/identity"
	modelsettings "github.com/keonho-kim/meca/internal/tui/settings/model"
	"github.com/rivo/uniseg"
	"strings"
	"testing"
)

func TestIdentityInfoMenuAndScroll(t *testing.T) {
	a := fixture(t)
	a.selection.Model = "models/bin/modelX"
	report := &identity.Report{Routed: true}
	report.Add("/model/info", "original", "Qwen/Qwen3.8-27B")
	report.Add("/model/info", "original", "ollama/gemma4")
	a.selection.Identity = report
	a.openModel()
	a.modelForm.Row = modelsettings.Information
	a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	if a.page != "model" || !a.modelForm.Details {
		t.Fatalf("page=%s", a.page)
	}
	a.render()
	if got := modelScreen(a); !strings.Contains(got, "복수") {
		t.Fatalf("identity state=%s", got)
	}
	a.screen.SetSize(36, 10)
	a.keyEvent(tcell.NewEventKey(tcell.KeyPgDn, "", 0))
	a.render()
	if a.modelForm.DetailsOffset == 0 {
		t.Fatal("identity evidence did not scroll")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.page != "model" {
		t.Fatalf("return page=%s", a.page)
	}
}

func TestIdentityNameBesideServingAlias(t *testing.T) {
	for _, alias := range []string{"team-coder", "코딩-모델"} {
		t.Run(alias, func(t *testing.T) {
			a := fixture(t)
			a.selection.Model = alias
			report := &identity.Report{}
			report.Add("server", "original", "/cache/models--Qwen--Qwen3.8-27B/snapshots/abc")
			a.selection.Identity = report
			a.openModel()
			a.render()
			if got := screenLine(a.screen, 1, 80); !strings.Contains(got, alias+"  Qwen/Qwen3.8-27B") {
				t.Fatalf("inline identity missing: %q", got)
			}
			_, aliasStyle, _ := a.screen.Get(0, 1)
			_, identityStyle, _ := a.screen.Get(uniseg.StringWidth(alias)+2, 1)
			if aliasStyle.HasDim() || !identityStyle.HasDim() {
				t.Fatal("alias and identity do not have distinct weights")
			}
			a.screen.SetSize(24, 10)
			a.render()
			if got := screenLine(a.screen, 1, 24); !strings.HasPrefix(got, alias+"  Qwen/") {
				t.Fatalf("narrow identity row=%q", got)
			}
		})
	}
}

func TestIdentityNameDoesNotClaimAmbiguousOrMissingOrigin(t *testing.T) {
	for _, report := range []*identity.Report{nil, {Routed: true}, {Incomplete: true}, {Unavailable: true}, {Facts: []identity.Fact{{Source: "server", Field: "family", Value: "qwen3"}}}} {
		a := fixture(t)
		a.selection.Model = "team-coder"
		a.selection.Identity = report
		a.openModel()
		a.render()
		if got := strings.TrimSpace(screenLine(a.screen, 1, 80)); got != "team-coder" {
			t.Fatalf("unidentified model claimed: %q", got)
		}
	}
}
