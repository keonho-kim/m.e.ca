// Purpose: Keep submitted images in user context and reject unsupported models before journal writes.
// Pattern: Request boundary regression tests.
// Usage: go test ./internal/agent/run -run TestSubmittedImages.
// Related: internal/agent/run/start.go, internal/agent/run/conversation.go.
package run

import (
	"github.com/keonho-kim/meca/internal/agent/media"
	"testing"
)

func TestSubmittedImagesReachModelAndRejectTextModel(t *testing.T) {
	image, err := media.New("clipboard.png", "image/png", "png", 1, 1, []byte("immutable compressed bytes"))
	if err != nil {
		t.Fatal(err)
	}
	for _, supported := range []bool{true, false} {
		g, j := &generator{replies: []string{"done"}}, &journal{}
		r := fixture(g, &executor{}, j)
		r.ImageInput, r.Images = supported, []media.Image{image}
		_, err := r.Run(t.Context(), "", approve, emit)
		if !supported {
			if err == nil || len(j.messages) != 0 || len(g.requests) != 0 {
				t.Fatal("unsupported model admitted images", err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, m := range g.requests[0].Messages {
			found = found || len(m.Images) == 1 && m.Images[0].Digest == image.Digest
		}
		if !found {
			t.Fatal("submitted image missing from model request")
		}
		if len(j.messages[0].Images) != 1 {
			t.Fatal("user image was not persisted")
		}
	}
}
