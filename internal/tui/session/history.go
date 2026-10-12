// Purpose: Load older ordered terminal records on demand while keeping model context separate.
// Pattern: Presentation projection over paged repository data.
// Usage: Called at restoration and when scrolling beyond the oldest loaded output.
// Related: internal/storage/sqlite/timeline.go, internal/tui/viewport/viewport.go.
package session

import (
	"fmt"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/storage/sqlite"
	"github.com/keonho-kim/v-code/internal/tui/viewport"
)

func (a *app) loadHistory(older bool) error {
	records, cursor, err := a.opts.Store.Timeline(a.ctx, a.opts.Session.ID, a.historyCursor, 100)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}
	a.historyCursor = cursor
	var blocks []viewport.Block
	for _, record := range records {
		blocks = append(blocks, a.recordBlocks(record)...)
	}
	if older {
		a.output.Prepend(blocks)
	} else {
		for _, block := range blocks {
			a.output.Append(block.Text, block.Thinking)
		}
	}
	return nil
}
func (a *app) recordBlocks(r sqlite.Record) []viewport.Block {
	if r.Kind == "message" {
		m := r.Message
		if m.Role == "user" {
			return []viewport.Block{{Text: "> " + a.opts.Vault.Redact(m.Content)}}
		}
		if m.Role == "assistant" && m.Planning {
			proposal, err := planning.ParsePlanProposal(m.Content)
			if err == nil {
				value := proposal.Objective
				for index, step := range proposal.Steps {
					value += fmt.Sprintf("\n%d. %s", index+1, step)
				}
				return []viewport.Block{{Text: a.opts.Vault.Redact(value)}}
			}
		}
		if m.Role == "assistant" && m.Content != "" {
			return []viewport.Block{{Text: a.opts.Vault.Redact(m.Content)}}
		}
		return nil
	}
	if cp := r.Event.Checkpoint; cp != nil && cp.Phase == "finished" {
		if cp.Status == run.Verified {
			return []viewport.Block{{Text: "Verification evidence: " + a.opts.Vault.Redact(cp.Evidence)}}
		}
		if cp.Status == run.Answered {
			return []viewport.Block{{Text: "Model answer; no tool execution."}}
		}
	}
	if r.Event.Kind == "thinking" {
		return []viewport.Block{{Text: a.opts.Vault.Redact(r.Event.Thinking), Thinking: true}}
	}
	if r.Event.Text != "" {
		return []viewport.Block{{Text: a.opts.Vault.Redact(r.Event.Text)}}
	}
	return nil
}
func (a *app) scroll(n, w, h int) {
	a.output.Scroll(n, w, h)
	if n < 0 && !a.output.Follow && a.output.Start(a.output.Lines(w), h) == 0 {
		a.fail(a.loadHistory(true))
	}
}
