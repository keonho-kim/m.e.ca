// Purpose: Keep unrelated observers progressing while another event is being redacted.
// Pattern: Controlled concurrency regression test.
// Usage: go test -race ./internal/serverapi -run TestEventProjectionIndependent.
// Related: internal/serverapi/event_cache.go.
package serverapi

import (
	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/commands/server"
	"sync"
	"testing"
	"time"
)

func TestEventProjectionIndependent(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	api := &API{RedactionVersion: func() int { return 0 }, Redact: func(value string) string {
		if value == "blocked" {
			once.Do(func() { close(started); <-release })
		}
		return value
	}}
	defer api.events.observe()()
	defer api.events.observe()()
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		api.projectEvent(server.Event{Cursor: "epoch:1", Record: record.Event{Kind: "progress", Text: "blocked"}})
	}()
	<-started
	other := make(chan struct{})
	go func() {
		defer close(other)
		api.projectEvent(server.Event{Cursor: "epoch:2", Record: record.Event{Kind: "progress", Text: "independent"}})
	}()
	select {
	case <-other:
	case <-time.After(time.Second):
		close(release)
		<-finished
		<-other
		t.Fatal("unrelated event held behind the global projection lock")
	}
	close(release)
	<-finished
}
