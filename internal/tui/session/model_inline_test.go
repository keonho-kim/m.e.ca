// Purpose: Keep model setting values, editing cursors and protected credentials on their original rows.
// Pattern: Virtual terminal interaction regression tests.
// Usage: go test ./internal/tui/session -run TestModelInline.
// Related: internal/tui/session/render_model.go, internal/tui/view/settings.go, internal/tui/session/input_region.go.
package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
	"github.com/keonho-kim/meca/internal/tui/locale"
	modelsettings "github.com/keonho-kim/meca/internal/tui/settings/model"
)

func TestModelInlineFieldStaysOnSelectedRow(t *testing.T) {
	for _, language := range []locale.Language{locale.English, locale.Korean} {
		for _, size := range [][2]int{{80, 24}, {36, 16}, {20, 10}} {
			for _, field := range []struct {
				name, value string
				row         int
			}{
				{"endpoint", "http://localhost:8000/v1", modelsettings.Endpoint},
				{"credential", "private-key-for-inline-test", modelsettings.Credential},
				{"model", "vendor/sample", modelsettings.Name},
			} {
				t.Run(fmt.Sprintf("%s/%v/%dx%d", field.name, language, size[0], size[1]), func(t *testing.T) {
					a := fixture(t)
					a.language = language
					a.screen.SetSize(size[0], size[1])
					a.openModel()
					f := a.modelForm
					f.Row = field.row
					f.Editor().Set(field.value)
					f.Models = []string{}
					a.render()
					before := selectedSettingRow(t, a)
					a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
					body := modelScreen(a)
					y := selectedSettingRow(t, a)
					if y != before {
						t.Fatal("entering the field moved its setting row")
					}
					tty, _ := a.screen.Tty()
					terminal := tty.(vt.MockTerm)
					if err := terminal.Drain(); err != nil {
						t.Fatal(err)
					}
					if int(terminal.Pos().Y) != y {
						t.Fatalf("cursor is on row %d, selected value is on row %d", terminal.Pos().Y, y)
					}
					if int(terminal.Pos().X) >= size[0] {
						t.Fatal("cursor escaped the value column")
					}
					if field.row == modelsettings.Credential {
						if strings.Contains(body, field.value) || a.inputRegion.editor != nil || !strings.Contains(screenLine(a.screen, y, size[0]), "*") {
							t.Fatal("inline credential was exposed or lost masking")
						}
					} else if a.inputRegion.y != y {
						t.Fatal("editable source mapping differs from its visible row")
					}
				})
			}
		}
	}
}

func selectedSettingRow(t *testing.T, a *app) int {
	t.Helper()
	w, h := a.screen.Size()
	selected := -1
	for y := 2; y < h-a.footerRows; y++ {
		if strings.HasPrefix(screenLine(a.screen, y, w), "> ") {
			if selected >= 0 {
				t.Fatal("setting has a duplicate input row")
			}
			selected = y
		}
	}
	if selected < 0 {
		t.Fatal("selected setting is not visible")
	}
	return selected
}

func TestModelInlineClickPositionsUnicodeCursor(t *testing.T) {
	a := fixture(t)
	a.openModel()
	f := a.modelForm
	f.Row, f.Editing = modelsettings.Name, true
	f.Models = []string{}
	f.Name.Set("A👩‍💻B")
	a.render()
	r := a.inputRegion
	if r.x <= 0 || r.contains(r.x-1, r.y) {
		t.Fatal("the setting label is part of the editable region")
	}
	a.mouse(tcell.NewEventMouse(r.x+1, r.y, tcell.Button1, 0))
	a.mouse(tcell.NewEventMouse(r.x+1, r.y, tcell.ButtonNone, 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "X", 0))
	if f.Name.String() != "AX👩‍💻B" {
		t.Fatal("inline click did not position the editor at the clicked grapheme")
	}
}

func TestModelInlineScrolledSelectionMapsToOriginal(t *testing.T) {
	a := fixture(t)
	a.openModel()
	f := a.modelForm
	f.Row, f.Editing = modelsettings.Endpoint, true
	f.Endpoint.Set("http://localhost:8000/v1")
	a.screen.SetSize(20, 10)
	a.render()
	r := a.inputRegion
	if r.window.Rows[0].Start <= 2 {
		t.Fatal("long value did not scroll within the column")
	}
	drag(a, r.x+r.width-3, r.y, r.x+r.width-2, r.y)
	if a.textSelection.Text() != "v1" {
		t.Fatalf("scrolled field copied %q instead of its visible suffix", a.textSelection.Text())
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyBackspace, "", 0))
	if f.Endpoint.String() != "http://localhost:8000/" {
		t.Fatal("scrolled selection deleted a different part of the source")
	}
}

func TestModelInlineCandidatesFollowValueColumn(t *testing.T) {
	a := fixture(t)
	a.openModel()
	f := a.modelForm
	f.Row, f.Editing = modelsettings.Name, true
	f.Name.Set("model")
	f.Models = []string{"model-one", "model-two"}
	f.RefreshHints()
	a.render()
	r := a.inputRegion
	w, _ := a.screen.Size()
	if line := screenLine(a.screen, r.y+1, w); !strings.HasPrefix(line[r.x:], "model-one") {
		t.Fatal("model candidates are detached from their edited value")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyDown, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
	if f.Name.String() != "model-two" || f.Editing || f.Confirm {
		t.Fatal("inline candidate selection changed its fill-only behavior")
	}
}
