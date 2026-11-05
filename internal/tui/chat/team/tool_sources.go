// Purpose: Refresh command summary state and masking while preserving original result sources.
// Pattern: Source-owned state and credential-revision invalidation.
// Usage: Team command clocks refresh summaries on status changes and remask after credential changes.
// Related: internal/tui/chat/team/tools.go, internal/tui/viewport/details.go.
package team

import (
	"sort"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/transcript"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func commandMaskVersion(f transcript.Formatter) int {
	if f.RedactionVersion != nil {
		return f.RedactionVersion()
	}
	return -1
}

func (b *toolDisplay) refreshStatus(run toolapi.CommandRun, f transcript.Formatter, output *viewport.View) {
	if b.status == run.Status {
		return
	}
	i := sort.Search(len(output.Blocks), func(i int) bool { return output.Blocks[i].ID >= b.body })
	if i >= len(output.Blocks) || output.Blocks[i].ID != b.body {
		return
	}
	value := output.Blocks[i]
	value.Summary = f.CommandSummary(run)
	if value.Details == "" {
		value.Text = value.Summary
	}
	output.Update(b.body, value)
	b.status = run.Status
}

func (b *toolDisplay) refreshMask(f transcript.Formatter, output *viewport.View) {
	version := commandMaskVersion(f)
	if version >= 0 && version == b.maskVersion {
		return
	}
	i := sort.Search(len(output.Blocks), func(i int) bool { return output.Blocks[i].ID >= b.body })
	if i >= len(output.Blocks) || output.Blocks[i].ID != b.body {
		return
	}
	value := output.Blocks[i]
	summary := f.Redact(value.Summary)
	if version >= 0 && value.DetailRef != nil {
		// Reload visible originals with the current mask instead of rescanning
		// a potentially large stale formatted body on the event loop.
		value.Text, value.Details, value.Changes = summary, "", nil
	} else {
		text := f.Redact(value.Text)
		if value.Details == value.Text {
			value.Details = text
		} else if value.Details != "" {
			value.Details = f.Redact(value.Details)
		}
		value.Text = text
	}
	value.Summary = summary
	output.Update(b.body, value)
	b.maskVersion = version
}
