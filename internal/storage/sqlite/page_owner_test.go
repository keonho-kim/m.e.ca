// Purpose: Prevent local payload identities from reusing another database's cached bytes.
// Pattern: Page-cache ownership regression test.
// Usage: go test ./internal/storage/sqlite -run TestPagePayloadKeepsStoreIdentity.
// Related: internal/storage/sqlite/page_payloads.go, internal/storage/sqlite/team_reference_page.go.
package sqlite

import (
	"context"
	"encoding/json"
	"testing"
)

func TestPagePayloadKeepsStoreIdentity(t *testing.T) {
	first, second := &Store{}, &Store{}
	p := pagePayloads{ctx: context.Background(), store: first}
	refs := []byte(`{"[\"content\"]":"1"}`)
	if err := p.include(2, p.store, refs); err != nil {
		t.Fatal(err)
	}
	if _, err := p.load("1", func(string) (json.RawMessage, error) { return []byte(`"first"`), nil }); err != nil {
		t.Fatal(err)
	}
	p.store = second
	if err := p.include(3, p.store, refs); err != nil {
		t.Fatal(err)
	}
	value, err := p.load("1", func(string) (json.RawMessage, error) { return []byte(`"second"`), nil })
	if err != nil || string(value) != `"second"` {
		t.Fatal("another store's payload was reused", string(value), err)
	}
}
