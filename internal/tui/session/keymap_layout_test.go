// Purpose: Verify translated primary and supporting hints occupy consistent footer rows.
// Pattern: Virtual terminal layout contract tests.
// Usage: go test ./internal/tui/session -run TestKeymapLayout.
// Related: internal/tui/session/keymap.go, internal/tui/view/keymap.go, internal/tui/locale/controls.go.
package session

import (
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/tui/locale"
	"github.com/keonho-kim/meca/internal/tui/preview"
	modelsettings "github.com/keonho-kim/meca/internal/tui/settings/model"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func TestKeymapLayoutPrimaryAndSupportingRows(t *testing.T) {
	for _, lang := range []locale.Language{locale.Korean, locale.English} {
		a := fixture(t)
		a.language = lang
		a.screen.SetSize(80, 24)
		a.worker = workerAgent
		a.output.AppendBlock(viewport.Block{Summary: "summary", Details: "details"})
		a.render()
		if a.footerRows != 3 {
			t.Fatalf("expected two keymap rows, got %d footer rows", a.footerRows)
		}
		primary := screenLine(a.screen, 22, 80)
		supporting := screenLine(a.screen, 23, 80)
		for _, key := range []string{"^C", "Esc", "Enter", "Tab", "!"} {
			if !strings.Contains(primary, key) {
				t.Fatalf("primary hint %q misplaced: %q", key, primary)
			}
		}
		for _, label := range []string{"줄바꿈", "입력 기록", "도구 접기", "스크롤"} {
			if !strings.Contains(supporting, lang.Text(label)) {
				t.Fatalf("supporting hint %q misplaced: %q", lang.Text(label), supporting)
			}
		}
	}
}

func TestKeymapLayoutClearAndPreviewLabels(t *testing.T) {
	for _, lang := range []locale.Language{locale.Korean, locale.English} {
		a := fixture(t)
		a.language = lang
		a.openModel()
		a.modelForm.Row, a.modelForm.Editing = modelsettings.Endpoint, true
		a.screen.SetSize(48, 24)
		a.render()
		if keys := footerKeyText(a); !strings.Contains(keys, lang.Text("Clear field")) || strings.Contains(keys, "←→") {
			t.Fatalf("field hints are ambiguous or redundant: %q", keys)
		}
		a.page = "files"
		a.filePreview = &preview.State{Full: true}
		if labels := keymapLabels(t, a); labels["Tab"] != lang.Text("분할 보기") {
			t.Fatal(labels)
		}
		a.filePreview.Full = false
		if labels := keymapLabels(t, a); labels["Enter"] != lang.Text("전체 화면") {
			t.Fatal(labels)
		}
	}
}
