// Purpose: Project the unsaved model form into localized setting rows and masked input.
// Pattern: Presentation snapshot builder.
// Usage: The model page renders this snapshot and publishes the active nonsecret input region.
// Related: internal/tui/session/model_form.go, internal/tui/view/settings.go, internal/tui/modelmenu/identity.go.
package session

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/tui/modelmenu"
	"github.com/keonho-kim/meca/internal/tui/view"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func (a *app) renderModel(w, h int) {
	if a.modelForm == nil {
		a.initializeModelForm()
	}
	f := a.modelForm
	options := f.options()
	providers := connection.Providers()
	providerNames := make([]string, len(providers))
	for i, p := range providers {
		providerNames[i] = p.Label
	}
	key := strings.Repeat("*", utf8.RuneCountInString(f.credential.String()))
	c, _ := f.connection()
	if key == "" && c.CredentialRef != "" {
		key = a.language.Text("Saved in OS credential store")
	}
	thinking := []string{}
	selectedThinking := -1
	if f.thinking == nil {
		thinking = append(thinking, "DEFAULT")
		selectedThinking = 0
	}
	if options.Enabled() {
		if f.thinking != nil && *f.thinking {
			selectedThinking = len(thinking)
		}
		thinking = append(thinking, "ON")
	}
	if options.Off {
		if f.thinking != nil && !*f.thinking {
			selectedThinking = len(thinking)
		}
		thinking = append(thinking, "OFF")
	}
	efforts := slices.Clone(options.Levels)
	if f.effort == "" {
		efforts = append([]string{"DEFAULT"}, efforts...)
	}
	thinkingHint := ""
	if options.Enabled() && !options.Off {
		thinkingHint = "(" + a.language.Text("OFF unavailable") + ")"
	}
	rows := []view.SettingRow{
		{Label: "Provider", Values: providerNames, Selected: f.provider},
		{Label: "Base URL", Value: a.opts.Vault.Redact(f.endpoint.String())},
		{Label: "API Key", Value: key},
		{Label: "Model ID", Value: a.opts.Vault.Redact(f.name.String())},
		{Label: "Thinking", Values: thinking, Selected: selectedThinking, Hint: thinkingHint},
		{Label: "Thinking effort", Values: efforts, Selected: slices.Index(efforts, f.effort)},
		{Label: "Save destination", Values: []string{a.language.Text("Current session"), a.language.Text("Global override")}, Selected: 0},
		{Label: "List models", Value: "→"},
		{Label: "Model information", Value: "→"},
		{Label: "Connection diagnostics", Value: "→"},
	}
	if f.effort == "" {
		rows[modelEffort].Selected = 0
	}
	if f.global {
		rows[modelDestination].Selected = 1
	}
	for i := range rows {
		rows[i].Label = a.language.Text(rows[i].Label)
	}
	display := view.SettingsDisplay{Title: a.language.Text("Model settings"), Language: a.language, Activity: a.activity(), Usage: a.displayTokens(), Rows: rows, Selected: f.row, Status: a.status, Confirm: f.confirm, Command: a.menuInput.String(), Subject: a.opts.Vault.Redact(f.name.String()), Annotation: modelmenu.IdentifiedName(f.selection(), a.opts.Vault.Redact)}
	if f.editing && !f.confirm {
		window := f.editor().FieldWindow(w, f.row == modelCredential)
		display.Field = &window
	}
	if f.editing && f.row == modelName && len(f.hints) > 0 {
		display.Candidates = view.Candidates{Language: a.language, Open: true, Labels: append([]string(nil), f.hints...), Selected: f.hint}
		for i, name := range display.Candidates.Labels {
			display.Candidates.Labels[i] = a.opts.Vault.Redact(name)
		}
	}
	if f.details {
		lines := viewport.Wrap(strings.Join(modelmenu.IdentityLines(f.selection(), a.opts.Config.ContextLength, a.opts.Vault.Redact, a.language), "\n"), w)
		f.detailsOffset = min(f.detailsOffset, max(0, len(lines)-1))
		display.Details = lines[f.detailsOffset:]
	}
	y := view.Settings(a.screen, display, w, h)
	if display.Field != nil && f.row != modelCredential && y >= 0 {
		a.inputRegion = inputRegion{editor: f.editor(), window: *display.Field, y: y, width: w, revision: f.editor().Revision()}
	}
}
