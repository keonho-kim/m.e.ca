// Purpose: Preserve each pending tool independently from notices and observed command lifetimes.
// Pattern: Event-loop-owned presentation state.
// Usage: Runtime notifications update pending rows; chatLayout projects them with command snapshots.
// Related: internal/tui/session/runtime_events.go, internal/tui/session/command_activity.go.
package session

import (
	"slices"
	"strings"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/transcript"
)

type pendingTool struct{ callID, text string }

type toolActivity struct {
	pending  []pendingTool
	commands []toolapi.CommandRun
	text     string
	dirty    bool
	views    map[string]transcript.CommandView
}

func toolCallID(event *runrecord.Event) string {
	if event.CallID != "" || event.Call == nil {
		return event.CallID
	}
	return event.Call.ID
}

func (a *app) startToolActivity(event *runrecord.Event) {
	if event.Call == nil || a.work.state.stopping() {
		return
	}
	a.chat.toolActivity.start(event, transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact})
}

func (t *toolActivity) start(event *runrecord.Event, formatter transcript.Formatter) {
	entry := pendingTool{callID: toolCallID(event), text: formatter.Tool(*event)}
	if index := slices.IndexFunc(t.pending, func(p pendingTool) bool { return p.callID == entry.callID }); index >= 0 {
		t.pending[index] = entry
	} else {
		t.pending = append(t.pending, entry)
	}
	t.dirty = true
}

func (a *app) finishToolActivity(event *runrecord.Event) {
	a.chat.toolActivity.finish(event)
	a.updateCommandActivity()
}

func (a *app) liveToolActivity() string {
	return a.chat.toolActivity.render(transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact, RedactionVersion: a.opts.Vault.RedactionVersion})
}

func (t *toolActivity) finish(event *runrecord.Event) {
	id := toolCallID(event)
	if index := slices.IndexFunc(t.pending, func(p pendingTool) bool { return p.callID == id }); index >= 0 {
		t.pending = slices.Delete(t.pending, index, index+1)
		t.dirty = true
	}

}

func (t *toolActivity) render(f transcript.Formatter) string {
	if !t.dirty {
		return t.text
	}
	var text strings.Builder
	for id := range t.views {
		if !slices.ContainsFunc(t.commands, func(run toolapi.CommandRun) bool { return run.ID == id && run.Status == "running" }) {
			delete(t.views, id)
		}
	}
	if len(t.views) == 0 {
		t.views = nil
	}
	for _, pending := range t.pending {
		// A running command snapshot replaces its start row without duplicating it.
		index := slices.IndexFunc(t.commands, func(run toolapi.CommandRun) bool {
			return pending.callID != "" && run.CallID == pending.callID && run.Status == "running"
		})
		if index >= 0 {
			text.WriteString(t.commandText(t.commands[index], f))
		} else {
			text.WriteString(pending.text)
		}
	}
	for _, run := range t.commands {
		if !slices.ContainsFunc(t.pending, func(p pendingTool) bool { return run.CallID != "" && p.callID == run.CallID }) {
			text.WriteString(t.commandText(run, f))
		}
	}
	t.text, t.dirty = text.String(), false
	return t.text
}

func (t *toolActivity) commandText(run toolapi.CommandRun, f transcript.Formatter) string {
	view := t.views[run.ID]
	text := view.Render(run, f)
	if run.Status == "running" && f.RedactionVersion != nil {
		if t.views == nil {
			t.views = make(map[string]transcript.CommandView)
		}
		t.views[run.ID] = view
	}
	return text
}

func (a *app) clearToolActivity() { a.chat.toolActivity = toolActivity{} }
