// Purpose: Verify shared event encodings never survive a credential revision or observer lifetime.
// Pattern: Secret-redaction and resource-lifecycle regression test.
// Usage: go test -race ./internal/serverapi -run TestEventProjectionCache.
// Related: internal/serverapi/event_cache.go.
package serverapi

import (
	"bytes"
	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"sync/atomic"
	"testing"
)

func TestEventProjectionCacheRevisionAndRelease(t *testing.T) {
	vault := &credentials.Vault{}
	var calls atomic.Int32
	api := &API{Redact: func(value string) string { calls.Add(1); return vault.Redact(value) }, RedactionVersion: vault.RedactionVersion}
	first, second := api.events.observe(), api.events.observe()
	defer second()
	event := server.Event{Cursor: "epoch:1", Record: record.Event{Kind: "progress", Text: "later-secret"}}
	notification := api.projectEvent(event)
	encoded, err := api.notificationJSON(notification)
	if err != nil {
		t.Fatal(err)
	}
	before := calls.Load()
	reused, err := api.notificationJSON(api.projectEvent(event))
	if err != nil || !bytes.Equal(encoded, reused) || calls.Load() != before {
		t.Fatal("identical observers repeated projection", err)
	}
	if err := vault.Remember("later-secret"); err != nil {
		t.Fatal(err)
	}
	masked, err := api.notificationJSON(api.projectEvent(event))
	if err != nil || bytes.Contains(masked, []byte("later-secret")) {
		t.Fatal("old cache crossed credential revision", err)
	}
	if event.Record.Text != "later-secret" {
		t.Fatal("cache changed the shared replay record")
	}
	first()
	if api.events.bytes != 0 || len(api.events.values) != 0 {
		t.Fatal("last shared observer retained event encodings")
	}
}
