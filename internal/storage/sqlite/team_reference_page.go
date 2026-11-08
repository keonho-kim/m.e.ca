// Purpose: Batch private event envelope reads and share one hydration budget across the page.
// Pattern: Page-scoped gather followed by ordered projection.
// Usage: Timeline and Export call prepare before resolving team_reference events.
// Related: internal/storage/sqlite/team_references.go, internal/storage/sqlite/page_payloads.go.
package sqlite

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/content"
	"github.com/keonho-kim/meca/internal/storage/history"
)

type privateRecord struct {
	record    storedRecord
	arguments []byte
}

func (p *teamPage) prepare(ctx context.Context, page []storedRecord) error {
	p.records = nil
	groups := map[string][]any{}
	refs := make(map[runrecord.EventReference]int)
	for i, r := range page {
		if r.kind != "event" || !bytes.Contains(r.body, []byte(`"team_reference"`)) {
			continue
		}
		var event struct {
			Kind      string                    `json:"kind"`
			Reference *runrecord.EventReference `json:"reference"`
		}
		if err := json.Unmarshal(r.body, &event); err != nil {
			return err
		}
		if event.Kind != "team_reference" {
			continue
		}
		if event.Reference == nil || event.Reference.Sequence < 1 {
			return faults.Errorf(diagcatalog.SqliteMissingPrivateEventReference)
		}
		groups[event.Reference.Session] = append(groups[event.Reference.Session], event.Reference.Sequence)
		refs[*event.Reference] = i
	}
	if len(refs) == 0 {
		return nil
	}
	p.records = make(map[runrecord.EventReference]privateRecord, len(refs))
	if p.payloads == nil {
		p.payloads = &pagePayloads{ctx: ctx}
	}
	for session, args := range groups {
		source, err := p.source(ctx, session)
		if err != nil {
			return err
		}
		maximum := content.MaxBytes
		if p.payloads.budget != nil && p.payloads.budget.limited {
			maximum = p.payloads.budget.remaining
		}
		rows, err := source.db.QueryContext(ctx, "SELECT sequence,CASE WHEN length(body)<=? THEN body END,refs,raw_bytes,display_args FROM tool_events WHERE sequence IN ("+placeholders(len(args))+")", append([]any{maximum}, args...)...)
		if err != nil {
			return err
		}
		for rows.Next() {
			var sequence int64
			entry := privateRecord{record: storedRecord{kind: "event"}}
			if err = rows.Scan(&sequence, &entry.record.body, &entry.record.refs, &entry.record.size, &entry.arguments); err != nil {
				break
			}
			if entry.record.body == nil {
				err = history.ErrReadLimit
				break
			}
			if p.payloads.budget != nil {
				if err = p.payloads.budget.take(len(entry.record.body)); err != nil {
					break
				}
			}
			ref := runrecord.EventReference{Session: session, Sequence: sequence}
			p.records[ref] = entry
			if err = p.payloads.include(refs[ref], source, entry.record.refs); err != nil {
				break
			}
		}
		if err = errors.Join(err, rows.Err(), rows.Close()); err != nil {
			return err
		}
	}
	if len(p.records) != len(refs) {
		return faults.Errorf(diagcatalog.SqliteReferencedPrivateEventIsMissing)
	}
	return nil
}
func compactToolFields(fields map[string]bool) map[string]bool {
	delete(fields, "output")
	delete(fields, "change_preview")
	delete(fields, "thinking")
	return fields
}
