// Purpose: Bind native retry identities to exact text and ordered immutable image content.
// Pattern: Pure input identity contract test.
// Usage: go test ./internal/commands/server -run TestInputDigest.
// Related: internal/commands/server/identity.go, internal/commands/server/tasks.go.
package server

import (
	"github.com/keonho-kim/meca/internal/agent/media"
	"testing"
)

func TestInputDigestBindsTextAndOrderedImages(t *testing.T) {
	a, _ := media.New("first.png", "image/png", "png", 1, 1, []byte("first"))
	b, _ := media.New("second.png", "image/png", "png", 1, 1, []byte("second"))
	want := InputDigest("inspect", []media.Image{a, b})
	if want == InputDigest("inspect", []media.Image{b, a}) || want == InputDigest("changed", []media.Image{a, b}) || want == InputDigest("inspect", []media.Image{a}) {
		t.Fatal("distinct input reused an admission digest")
	}
	a.Path = "new-upload-name.png"
	if want != InputDigest("inspect", []media.Image{a, b}) {
		t.Fatal("upload label changed identical image intent")
	}
}
