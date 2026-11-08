// Purpose: Discover skill completions asynchronously while preserving the current composer draft.
// Pattern: Single-read latest-query presentation adapter.
// Usage: refreshHints requests metadata suggestions and the event loop applies matching results.
// Related: internal/tui/session/hints.go, internal/tui/session/skills.go.
package session

import (
	"context"
	"errors"
	"strings"

	"github.com/keonho-kim/meca/internal/commands"
)

func skillCompletion(text string) string {
	start := strings.LastIndexAny(text, " \t\n") + 1
	token := text[start:]
	if !strings.HasPrefix(token, "#") || strings.ContainsAny(token, "`\"'\\/") {
		return ""
	}
	return token
}

func (a *app) requestSkillHints(query string) {
	if a.skills.hintBusy {
		return
	}
	reader := a.skillReader()
	if reader == nil {
		return
	}
	a.skills.hintBusy = true
	a.skills.hintJob++
	job := a.skills.hintJob
	ctx, cancel := context.WithCancel(a.ctx)
	a.skills.hintCancel = cancel
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		items, err := reader.Suggestions(ctx, strings.TrimPrefix(query, "#"))
		_ = a.post(a.ctx, notice{Skills: &skillNotice{query: query, suggestions: items, err: err, job: job}})
	}()
}

func (a *app) receiveSkillHints(n *skillNotice) {
	query := skillCompletion(a.compose.editor.String())
	if a.page != "chat" || query == "" || a.compose.editor.Cursor != a.compose.editor.Len() || a.paste || a.compose.queued.Edit != nil {
		return
	}
	if query != n.query {
		a.requestSkillHints(query)
		return
	}
	if n.err != nil {
		if errors.Is(n.err, context.Canceled) {
			return
		}
		a.fail(n.err)
		return
	}
	a.compose.commandHints = nil
	for _, m := range n.suggestions {
		name := m.Name
		if strings.Contains(query, ":") {
			name = m.ID
		}
		a.compose.commandHints = append(a.compose.commandHints, commands.Suggestion{Name: "#" + name, Description: m.Description + " [" + m.Source + "]"})
	}
	a.compose.commandIndex = 0
	a.compose.commandOpen = len(a.compose.commandHints) > 0
}
