// Purpose: Prepare a bounded journal prefix before reserving SQLite's single writer.
// Pattern: Operation-owned record encoder with optimistic immutable-payload lookups.
// Usage: Commit validates first, prepares up to its buffer budget, then atomically applies records.
// Related: internal/storage/sqlite/commit.go, internal/storage/sqlite/prepared_payload.go.
package sqlite

import (
	"context"
	"database/sql"

	"unsafe"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const journalPrepareBytes = 4 << 20

type payloadLookup interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type preparedRecord struct {
	source                 runrecord.Entry
	body, refs, display    []byte
	referenceSlots         map[string]string
	size, memoryBytes      int
	memoryRole, memoryText string
	memorySource           int64
	payloads               []*preparedPayload
	state                  *preparedState
}

type journalEncoder struct {
	store      *Store
	ctx        context.Context
	lookup     payloadLookup
	payloads   map[string]*preparedPayload
	referenced []*preparedPayload
	retained   int
}

func checkJournal(ctx context.Context, records []runrecord.Entry) error {
	for _, entry := range records {
		if err := ctx.Err(); err != nil {
			return err
		}
		switch {
		case entry.Message != nil && entry.Event == nil:
			if err := entry.Message.Check(); err != nil {
				return err
			}
		case entry.Event != nil && entry.Message == nil:
			if err := entry.Event.Check(); err != nil {
				return err
			}
		default:
			return faults.Errorf(diagcatalog.SqliteJournalRecordRequiresExactlyOnePayload)
		}
	}
	return ctx.Err()
}

func (s *Store) prepareJournal(ctx context.Context, records []runrecord.Entry) ([]preparedRecord, error) {
	encoder := journalEncoder{store: s, ctx: ctx, lookup: s.db}
	entryBytes := int(unsafe.Sizeof(preparedRecord{}))
	prepared := make([]preparedRecord, 0, min(len(records), 32))
	encoder.retained = cap(prepared) * entryBytes
	for _, entry := range records {
		// One accepted record may exceed the preparation budget. Never retain
		// another record after it; remaining entries use the same encoder in-tx.
		if encoder.retained >= journalPrepareBytes {
			break
		}
		if len(prepared) == cap(prepared) {
			extra := min(max(1, cap(prepared)), len(records)-len(prepared))
			if encoder.retained+extra*entryBytes > journalPrepareBytes {
				break
			}
			next := make([]preparedRecord, len(prepared), cap(prepared)+extra)
			copy(next, prepared)
			prepared = next
			encoder.retained += extra * entryBytes
		}
		value, err := encoder.prepare(entry)
		if err != nil {
			return nil, err
		}
		prepared = append(prepared, value)
	}
	return prepared, nil
}

func (p *journalEncoder) prepare(entry runrecord.Entry) (preparedRecord, error) {
	if err := p.ctx.Err(); err != nil {
		return preparedRecord{}, err
	}
	p.referenced = nil
	value := preparedRecord{source: entry}
	var err error
	if entry.Message != nil {
		err = p.message(&value)
	} else {
		err = p.event(&value)
	}
	if err != nil {
		return preparedRecord{}, err
	}
	value.payloads = p.referenced
	p.referenced = nil
	p.retained += len(value.body) + len(value.refs) + len(value.display) + len(value.memoryText) + cap(value.payloads)*int(unsafe.Sizeof((*preparedPayload)(nil)))
	for path, slot := range value.referenceSlots {
		p.retained += 96 + len(path) + len(slot)
	}
	if value.state != nil {
		p.retained += len(value.state.body) + len(value.state.id) + int(unsafe.Sizeof(*value.state))
	}
	return value, nil
}

func (value *preparedRecord) write(ctx context.Context, store *Store, tx *sql.Tx, session, now string, receipt *journalReceipt) error {
	for _, payload := range value.payloads {
		if err := payload.write(ctx, tx); err != nil {
			return err
		}
	}
	if err := value.localRefs(); err != nil {
		return err
	}
	if value.source.Message != nil {
		return store.writeMessage(ctx, tx, session, value, now, receipt)
	}
	return store.writeEvent(ctx, tx, session, value, now, receipt)
}
