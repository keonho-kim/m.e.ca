// Purpose: Keep command metadata reads from blocking peer registration and removal.
// Pattern: Deterministic concurrent source-lifetime regression test.
// Usage: Service snapshots source ownership before calling close-safe metadata readers.
// Related: internal/commands/teamrequest/activity.go, internal/commands/teamrequest/service.go.
package teamrequest

import (
	"testing"
	"time"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type waitingCommandSource struct{ entered, release chan struct{} }

func (s waitingCommandSource) CommandActivity() []toolapi.CommandRun {
	close(s.entered)
	<-s.release
	return []toolapi.CommandRun{{ID: "command", Status: "running"}}
}

func TestCommandSourceReadDoesNotHoldRegistryLock(t *testing.T) {
	source := waitingCommandSource{entered: make(chan struct{}), release: make(chan struct{})}
	service := &Service{}
	service.track("Hana", source)
	done := make(chan struct{})
	go func() { service.CommandActivity(); close(done) }()
	<-source.entered
	removed := make(chan struct{})
	go func() { service.track("Hana", nil); close(removed) }()
	select {
	case <-removed:
	case <-time.After(5 * time.Second):
		close(source.release)
		<-done
		t.Fatal("metadata reader retained the service registry lock")
	}
	close(source.release)
	<-done
	if service.active != nil {
		t.Fatal("removed source remained registered")
	}
}
