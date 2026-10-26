// Purpose: Project model settings into localized rows without owning the screen or active worker.
// Pattern: Settings presentation projection.
// Usage: The session renderer adds runtime activity, usage and command status before painting.
// Related: internal/tui/settings/model/form.go, internal/tui/view/settings.go.
package model

import (
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"github.com/keonho-kim/meca/internal/tui/view"
	"github.com/keonho-kim/meca/internal/tui/viewport"
	"slices"
	"strings"
	"unicode/utf8"
)

func (f *Form) Display(w, contextLength int, language locale.Language, redact func(string) string) view.SettingsDisplay {
	options := f.Options()
	providers := connection.Providers()
	providerNames := make([]string, len(providers))
	for i, p := range providers {
		providerNames[i] = p.Label
	}
	key := strings.Repeat("*", utf8.RuneCountInString(f.Credential.String()))
	c, _ := f.Connection()
	if key == "" && c.CredentialRef != "" {
		key = language.Text("Saved in OS credential store")
	}
	thinking := []string{}
	selectedThinking := -1
	if f.Thinking == nil {
		thinking = append(thinking, "DEFAULT")
		selectedThinking = 0
	}
	if options.Enabled() {
		if f.Thinking != nil && *f.Thinking {
			selectedThinking = len(thinking)
		}
		thinking = append(thinking, "ON")
	}
	if options.Off {
		if f.Thinking != nil && !*f.Thinking {
			selectedThinking = len(thinking)
		}
		thinking = append(thinking, "OFF")
	}
	efforts := slices.Clone(options.Levels)
	if f.Effort == "" {
		efforts = append([]string{"DEFAULT"}, efforts...)
	}
	thinkingHint := ""
	if options.Enabled() && !options.Off {
		thinkingHint = "(" + language.Text("OFF unavailable") + ")"
	}
	rows := []view.SettingRow{
		{Label: "Provider", Values: providerNames, Selected: f.Provider},
		{Label: "Base URL", Value: redact(f.Endpoint.String())},
		{Label: "API Key", Value: key},
		{Label: "Model ID", Value: redact(f.Name.String())},
		{Label: "Thinking", Values: thinking, Selected: selectedThinking, Hint: thinkingHint},
		{Label: "Thinking effort", Values: efforts, Selected: slices.Index(efforts, f.Effort)},
		{Label: "Save destination", Values: []string{language.Text("Current session"), language.Text("Global override")}, Selected: 0},
		{Label: "Model information", Value: "→"},
		{Label: "Connection diagnostics", Value: "→"},
	}
	if f.Effort == "" {
		rows[Effort].Selected = 0
	}
	if f.Global {
		rows[Destination].Selected = 1
	}
	for i := range rows {
		rows[i].Label = language.Text(rows[i].Label)
	}
	display := view.SettingsDisplay{Title: language.Text("Model settings"), Language: language, Rows: rows, Selected: f.Row, Confirm: f.Confirm, Subject: redact(f.Name.String()), Annotation: identifiedName(f.Selection(), redact)}
	fieldX := view.SettingsValueColumn(w)
	if f.Editing && !f.Confirm {
		window := f.Editor().FieldWindow(w-fieldX, f.Row == Credential)
		display.Field = &window
	}
	if f.Editing && f.Row == Name && len(f.Hints) > 0 {
		display.Candidates = view.Candidates{Language: language, Open: true, Labels: append([]string(nil), f.Hints...), Selected: f.Hint}
		for i, name := range display.Candidates.Labels {
			display.Candidates.Labels[i] = redact(name)
		}
	}
	if f.Details {
		lines := viewport.Wrap(strings.Join(identityLines(f.Selection(), contextLength, redact, language), "\n"), w)
		f.DetailsOffset = min(f.DetailsOffset, max(0, len(lines)-1))
		display.Details = lines[f.DetailsOffset:]
	}
	return display
}
