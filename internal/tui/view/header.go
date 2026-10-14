// Purpose: Render model, enabled Thinking, automatic approval, Scope and Plan in the status line.
// Pattern: Presentation renderer.
// Usage: The session renderer supplies current values without sharing mutable task state.
// Related: internal/tui/session/render.go, internal/policy/policy.go, internal/tui/view/cells.go.
package view

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/policy"
	"github.com/rivo/uniseg"
	"strings"
)

func Header(screen tcell.Screen, model string, thinking bool, effort string, settings policy.Settings, planning bool, y, w int) {
	if model == "" {
		model = "no-model"
	}
	suffix := ""
	if thinking {
		suffix = " Thinking"
		if effort != "" {
			suffix += " " + strings.ToUpper(effort)
		}
	}
	if settings.Permission == policy.AllowAll {
		suffix += " │ 자동 승인"
	}
	scope := "프로젝트"
	if settings.Scope == policy.System {
		scope = "전체 시스템"
	}
	suffix += " │ " + scope
	if planning {
		suffix += " │ PLAN"
	}
	maxModel := max(1, w-uniseg.StringWidth(suffix))
	Text(screen, 0, y, maxModel, model, tcell.StyleDefault)
	Text(screen, min(uniseg.StringWidth(model), maxModel), y, w, suffix, tcell.StyleDefault)
}
