// Purpose: Render monochrome settings with explicit selected values and save confirmation.
// Pattern: Read-only terminal projection.
// Usage: Model and language screens supply read-only setting snapshots.
// Related: internal/tui/session/render_model.go, internal/tui/session/render_language.go, internal/tui/view/input.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/tui/input"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"github.com/rivo/uniseg"
)

type SettingRow struct {
	Label, Value, Hint string
	Values             []string
	Selected           int
}

type SettingsDisplay struct {
	Title, Subject, Annotation string
	Language                   locale.Language
	Activity                   Activity
	Usage                      modelapi.TokenUsage
	Rows                       []SettingRow
	Selected                   int
	Field                      *input.Window
	Candidates                 Candidates
	Details                    []string
	Status, Command            string
	Confirm                    bool
}

// Settings returns the full-width editor row for clipboard source mapping.
func Settings(screen tcell.Screen, d SettingsDisplay, w, h int) int {
	if h < 4 {
		Text(screen, 0, 0, w, d.Language.Text("Resize terminal"), tcell.StyleDefault)
		return -1
	}
	topLine(screen, d.Title, d.Usage, w, d.Activity)
	Text(screen, 0, 1, w, d.Subject, tcell.StyleDefault)
	x := uniseg.StringWidth(d.Subject) + 2
	if x < w {
		Text(screen, x, 1, w-x, d.Annotation, tcell.StyleDefault.Dim(true))
	}
	space := max(1, h-3)
	if d.Field != nil {
		space--
	}
	reserved := 0
	if d.Candidates.Open {
		reserved = min(5, max(0, space-1))
	} else if len(d.Details) > 0 {
		reserved = min(4, max(0, space/2))
	}
	if d.Confirm {
		reserved = max(reserved, 2)
	}
	visible := max(1, space-reserved)
	first := max(0, d.Selected-visible+1)
	y, fieldY := 2, -1
	labelWidth := min(24, max(8, w/3))
	for index := first; index < min(len(d.Rows), first+visible); index++ {
		row := d.Rows[index]
		prefix := "  "
		if index == d.Selected {
			prefix = "> "
		}
		Text(screen, 0, y, labelWidth, prefix+row.Label, SelectionStyle(index == d.Selected))
		x := labelWidth + 1
		if row.Values == nil {
			Text(screen, x, y, max(0, w-x), row.Value, SelectionStyle(index == d.Selected))
		} else {
			start := 0
			for i := 0; i < row.Selected; i++ {
				if uniseg.StringWidth(joinValues(row.Values[start:row.Selected+1])) > w-x {
					start++
				}
			}
			for i := start; i < len(row.Values) && x < w; i++ {
				if i > start {
					Text(screen, x, y, max(0, w-x), " / ", tcell.StyleDefault.Dim(true))
					x += 3
				}
				Text(screen, x, y, max(0, w-x), row.Values[i], SelectionStyle(i == row.Selected))
				x += uniseg.StringWidth(row.Values[i])
			}
			if row.Hint != "" {
				Text(screen, x+1, y, max(0, w-x-1), row.Hint, tcell.StyleDefault.Dim(true))
			}
		}
		y++
		if d.Field != nil && index == d.Selected {
			fieldY = y
			Input(screen, *d.Field, y, w)
			y++
		}
	}
	if !d.Confirm {
		if d.Candidates.Open {
			if y+reserved+1 < h-1 {
				y++
			}
			candidates(screen, d.Candidates, y, w, h-1)
		} else {
			for _, line := range d.Details {
				if y >= h-2 {
					break
				}
				Text(screen, 0, y, w, line, tcell.StyleDefault.Dim(true))
				y++
			}
		}
	}
	if d.Confirm {
		Text(screen, 0, max(1, h-3), w, d.Language.Text("Save current settings?"), tcell.StyleDefault.Bold(true))
		Text(screen, 0, max(2, h-2), w, "y "+d.Language.Text("Save")+"   n "+d.Language.Text("Discard")+"   Esc "+d.Language.Text("Keep editing"), tcell.StyleDefault)
		screen.HideCursor()
	}
	Text(screen, 0, h-1, w, d.Status, tcell.StyleDefault)
	if d.Command != "" {
		Text(screen, 0, h-1, w, "> "+d.Command, tcell.StyleDefault)
	}
	return fieldY
}

func joinValues(values []string) string {
	text := ""
	for i, value := range values {
		if i > 0 {
			text += " / "
		}
		text += value
	}
	return text
}
