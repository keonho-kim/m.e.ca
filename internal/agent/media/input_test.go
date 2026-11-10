// Purpose: Verify enlarged image acceptance and bounded request collections.
// Pattern: Boundary contract tests.
// Usage: go test ./internal/agent/media.
// Related: internal/agent/media/image.go, internal/agent/media/input.go.
package media

import "testing"

func TestLargeImageInput(t *testing.T) {
	image, err := New("attached.png", "image/png", "png", 8192, 8192, make([]byte, 33<<20))
	if err != nil {
		t.Fatalf("64M-pixel input below 64MiB rejected: %v", err)
	}
	if err := CheckInput([]Image{image}); err != nil {
		t.Fatal(err)
	}
	images := make([]Image, MaxImages+1)
	if CheckInput(images) == nil {
		t.Fatal("unbounded image count")
	}
	if _, err := New("attached.png", "image/png", "png", 8193, 8192, image.Data); err == nil {
		t.Fatal("excess pixels accepted")
	}
}
