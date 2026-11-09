// Purpose: Prevent local payload identities from reusing another database's cached bytes.
// Pattern: Page-cache ownership regression test.
// Usage: go test ./internal/storage/sqlite -run TestPagePayloadKeepsStoreIdentity.
// Related: internal/storage/sqlite/page_payloads.go, internal/storage/sqlite/team_reference_page.go.
package sqlite

import (
	"context"
	"encoding/json"
	"testing"

	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

func TestPagePayloadKeepsStoreIdentity(t *testing.T) {
	first := &Store{session: sessionstate.Session{ID: "first"}}
	second := &Store{session: sessionstate.Session{ID: "second"}}
	p := pagePayloads{ctx: context.Background(), store: first}
	refs := []byte(`{"[\"content\"]":"1"}`)
	if err := p.include(2, p.store, refs); err != nil {
		t.Fatal(err)
	}
	if _, err := p.load(first.session.ID, "1", func(string) (json.RawMessage, error) { return []byte(`"first"`), nil }); err != nil {
		t.Fatal(err)
	}
	p.store = second
	if err := p.include(3, p.store, refs); err != nil {
		t.Fatal(err)
	}
	value, err := p.load(second.session.ID, "1", func(string) (json.RawMessage, error) { return []byte(`"second"`), nil })
	if err != nil || string(value) != `"second"` {
		t.Fatal("another store's payload was reused", string(value), err)
	}
}

func TestPagePayloadKeepsIdentityAfterHandleReplacement(t *testing.T) {
	first := &Store{session: sessionstate.Session{ID: "same-session"}}
	p := pagePayloads{ctx: context.Background(), store: first}
	if err := p.include(2, first, []byte(`{"[\"content\"]":"1"}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := p.load(first.session.ID, "1", func(string) (json.RawMessage, error) { return []byte(`"original"`), nil }); err != nil {
		t.Fatal(err)
	}
	reopened := &Store{session: first.session}
	value, err := p.load(reopened.session.ID, "1", func(string) (json.RawMessage, error) {
		t.Fatal("reopened source lost page-local reuse")
		return nil, nil
	})
	if err != nil || string(value) != `"original"` {
		t.Fatal(string(value), err)
	}
}
