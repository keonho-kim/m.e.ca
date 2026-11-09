// Purpose: Keep allocation-free image reference hashing byte-identical to the canonical stored JSON.
// Pattern: Encoding contract regression test.
// Usage: go test ./internal/storage/sqlite -run TestTeamImageIdentity.
// Related: internal/storage/sqlite/team_image_identity.go, internal/storage/sqlite/team_images.go.
package sqlite

import (
	"encoding/json"
	"github.com/keonho-kim/meca/internal/agent/media"
	"github.com/keonho-kim/meca/internal/storage/content"
	"testing"
)

func TestTeamImageIdentityMatchesCanonicalEncoding(t *testing.T) {
	image, err := media.New("한글<&>.png", "image/png", "png", 10, 10, []byte("five!"))
	if err != nil {
		t.Fatal(err)
	}
	image.Representative = true
	for _, images := range [][]media.Image{nil, {}, {image, image}} {
		body, err := json.Marshal(images)
		if err != nil {
			t.Fatal(err)
		}
		id, size, err := teamImageIdentity(images)
		if err != nil || id != content.ID(body) || size != len(body) {
			t.Fatal("reference identity changed", id, size, err)
		}
	}
}
