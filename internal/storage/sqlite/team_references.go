// Purpose: Resolve room tool references from private originals without duplicating captured payloads.
// Pattern: Bounded page-scoped hydration adapter.
// Usage: Timeline and export resolve team_reference events after closing their page query.
// Related: internal/storage/sqlite/timeline.go, internal/storage/sqlite/team_members.go.
package sqlite

import (
	"context"
	"encoding/json"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	recordcontent "github.com/keonho-kim/meca/internal/storage/content"
	"github.com/keonho-kim/meca/internal/storage/history"
)

type teamPage struct {
	parent   *Store
	stores   map[string]*Store
	records  map[runrecord.EventReference]privateRecord
	payloads *pagePayloads
	compact  bool
	index    int
}

func (p *teamPage) resolve(ctx context.Context, event runrecord.Event, display bool) (runrecord.Event, error) {
	if event.Reference == nil || event.Reference.Sequence < 1 {
		return event, faults.Errorf(diagcatalog.SqliteMissingPrivateEventReference)
	}
	entry, ok := p.records[*event.Reference]
	if !ok {
		source, err := p.source(ctx, event.Reference.Session)
		if err != nil {
			return event, err
		}
		entry.record.kind = "event"
		err = source.db.QueryRowContext(ctx, "SELECT body,refs,raw_bytes,display_args FROM tool_events WHERE sequence=?", event.Reference.Sequence).Scan(&entry.record.body, &entry.record.refs, &entry.record.size, &entry.arguments)
		if err != nil {
			return event, err
		}
	}
	record, arguments := entry.record, entry.arguments
	var fields map[string]bool
	if display {
		fields = history.DisplayFields("event")
	}
	if p.compact {
		fields = compactToolFields(fields)
	}
	var body []byte
	var err error
	read := func(id string) (json.RawMessage, error) {
		source, err := p.source(ctx, event.Reference.Session)
		if err != nil {
			return nil, err
		}
		if p.payloads != nil && p.payloads.budget != nil {
			return source.readPayload(ctx, id, p.payloads.budget)
		}
		return source.payload(ctx, id)
	}
	if p.payloads != nil {
		body, err = p.payloads.unpackWith(p.index, event.Reference.Session, record, fields, read)
	} else {
		body, err = recordcontent.Restore(record.body, record.refs, fields, read)
	}
	if err != nil {
		return event, err
	}
	var original runrecord.Event
	if display {
		projected, err := history.Display(0, "event", body, record.body, arguments)
		if err != nil {
			return event, err
		}
		original = projected.Event
	} else if err = json.Unmarshal(body, &original); err != nil {
		return event, err
	}
	original.Agent = event.Agent
	if display {
		original.Reference = event.Reference
	}
	return original, nil
}
