// Purpose: Preserve foldable pending tools independently from notices and observed command lifetimes.
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

type pendingTool struct {
	callID, kind, summary, details string
	// The immutable call borrows argument bytes until completion; rebuilding
	// summaries must redact complete values before truncating their targets.
	call toolapi.Call
}

type toolActivity struct {
	pending     []pendingTool
	commands    []toolapi.CommandRun
	text        string
	dirty       bool
	collapsed   bool
	maskVersion int
	views       map[string]transcript.CommandView
}

func toolCallID(event *runrecord.Event) string {
	if event.Call != nil && event.Call.ExecutionID != "" {
		return event.Call.ExecutionID
	}
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
	if id := transcript.PendingCommand(*event.Call); id != "" && slices.ContainsFunc(t.commands, func(run toolapi.CommandRun) bool { return run.ID == id }) {
		return
	}
	block := formatter.ToolBlock(*event, false)
	entry := pendingTool{callID: toolCallID(event), kind: event.Kind, call: *event.Call, summary: block.Summary, details: block.Details}
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
	return a.chat.toolActivity.render(transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact, RedactionVersion: a.opts.Vault.RedactionVersion}, a.chat.output.Collapsed)
}

func (t *toolActivity) finish(event *runrecord.Event) {
	id := toolCallID(event)
	if index := slices.IndexFunc(t.pending, func(p pendingTool) bool { return p.callID == id }); index >= 0 {
		t.pending = slices.Delete(t.pending, index, index+1)
		t.dirty = true
	}
}

func (t *toolActivity) render(f transcript.Formatter, collapsed bool) string {
	version := -1
	if f.RedactionVersion != nil {
		version = f.RedactionVersion()
	}
	if !t.dirty && t.collapsed == collapsed && version >= 0 && t.maskVersion == version {
		return t.text
	}
	for i := range t.pending {
		pending := &t.pending[i]
		if version < 0 || t.maskVersion != version {
			block := f.ToolBlock(runrecord.Event{Kind: pending.kind, Call: &pending.call}, false)
			pending.summary, pending.details = block.Summary, ""
		}
		if collapsed {
			pending.details = ""
		} else if pending.details == "" {
			block := f.ToolBlock(runrecord.Event{Kind: pending.kind, Call: &pending.call}, true)
			pending.summary, pending.details = block.Summary, block.Details
		}
	}
	var text strings.Builder
	for id := range t.views {
		if !slices.ContainsFunc(t.commands, func(run toolapi.CommandRun) bool { return run.ID == id && toolapi.CommandActive(run.Status) }) {
			delete(t.views, id)
		}
	}
	if len(t.views) == 0 {
		t.views = nil
	}
	for _, pending := range t.pending {
		// A running command snapshot replaces its start row without duplicating it.
		index := slices.IndexFunc(t.commands, func(run toolapi.CommandRun) bool {
			return pending.callID != "" && commandExecutionKey(run) == pending.callID && toolapi.CommandActive(run.Status)
		})
		if index >= 0 {
			text.WriteString(t.commandText(t.commands[index], f, collapsed))
		} else {
			body := pending.details
			if collapsed {
				body = pending.summary
			}
			text.WriteString(body)
		}
	}
	for _, run := range t.commands {
		if !slices.ContainsFunc(t.pending, func(p pendingTool) bool {
			return commandExecutionKey(run) != "" && p.callID == commandExecutionKey(run)
		}) {
			text.WriteString(t.commandText(run, f, collapsed))
		}
	}
	t.text, t.dirty = text.String(), false
	t.collapsed, t.maskVersion = collapsed, version
	return t.text
}

func commandExecutionKey(run toolapi.CommandRun) string {
	if run.ExecutionID != "" {
		return run.ExecutionID
	}
	return run.CallID
}

func (t *toolActivity) commandText(run toolapi.CommandRun, f transcript.Formatter, collapsed bool) string {
	view := t.views[run.ID]
	text := view.Render(run, f, collapsed)
	if toolapi.CommandActive(run.Status) && f.RedactionVersion != nil {
		if t.views == nil {
			t.views = make(map[string]transcript.CommandView)
		}
		t.views[run.ID] = view
	}
	return text
}

func (t *toolActivity) hasDetails() bool { return len(t.pending) > 0 || len(t.commands) > 0 }

func (a *app) clearToolActivity() { a.chat.toolActivity = toolActivity{} }
