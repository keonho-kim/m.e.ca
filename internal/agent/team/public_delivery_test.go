// Purpose: Keep durable public progress available while a terminal consumer is slow.
// Pattern: Controlled presentation barrier and ordered publication regression.
// Usage: go test -race ./internal/agent/team -run TestSlowPublicDelivery.
// Related: internal/agent/team/publication.go, internal/agent/team/public_delivery.go.
package team

import (
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"testing"
)

func TestSlowPublicDeliveryDoesNotHoldPublicationLock(t *testing.T) {

	r := testRoom(t)
	blocked, release := make(chan struct{}), make(chan struct{})
	var shown []string
	r.options.Emit = func(event runrecord.Event) error {
		if event.Text == "first" {
			close(blocked)
			<-release
		}
		shown = append(shown, event.Text)
		return nil
	}
	done := make(chan error, 2)
	go func() { _, err := r.publish(channel.Message{From: "Hana", Text: "first"}); done <- err }()
	<-blocked
	if !r.logMu.TryLock() {
		close(release)
		<-done
		t.Fatal("slow UI delivery held the durable publication lock")
	}
	r.logMu.Unlock()
	go func() { _, err := r.publish(channel.Message{From: "Ren", Text: "second"}); done <- err }()
	close(release)
	for range 2 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	r.mu.Lock()
	through := r.latestPublic
	r.mu.Unlock()
	if through != 2 || len(shown) != 2 || shown[0] != "first" || shown[1] != "second" {
		t.Fatalf("publication blocked or delivery reordered: cursor=%d shown=%v", through, shown)
	}
}
