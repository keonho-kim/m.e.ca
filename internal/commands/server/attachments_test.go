// Purpose: Verify expiring upload ownership and shutdown joins without retaining inactive image bytes.
// Pattern: Deterministic virtual-clock lifecycle test.
// Usage: go test -race ./internal/commands/server -run TestImageAttachmentExpiry.
// Related: internal/commands/server/attachments.go, internal/commands/server/lifecycle.go.
package server

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/media"
	"testing"
	"testing/synctest"
	"time"
)

func TestImageAttachmentExpiryAndCloseReleaseOwnership(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		s := &Service{ctx: ctx, cancel: cancel, closeDone: make(chan struct{}), feed: newEventLog()}
		image, _ := media.New("uploaded.png", "image/png", "png", 1, 1, []byte("immutable image"))
		id, err := s.StageImage(image)
		if err != nil {
			t.Fatal(err)
		}
		borrowed, err := s.ImageAttachments([]string{id})
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(AttachmentLifetime)
		synctest.Wait()
		if len(s.attachments) != 0 {
			t.Fatal("expired image retained")
		}
		if _, err := s.ImageAttachments([]string{id}); err == nil {
			t.Fatal("expired reference accepted")
		}
		if string(borrowed[0].Data) != "immutable image" {
			t.Fatal("expiry invalidated borrowed pixels")
		}
		if _, err := s.StageImage(image); err != nil {
			t.Fatal(err)
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
		if len(s.attachments) != 0 {
			t.Fatal("close retained pending image")
		}
	})
}
