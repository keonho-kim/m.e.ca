// Purpose: Verify WebP snapshots and representative animation frames without external runtime codecs.
// Pattern: Authored raster-fixture regression tests.
// Usage: go test ./internal/tools/local/imagefile -run TestWebP.
// Related: internal/tools/local/imagefile/webp.go, internal/tools/local/imagefile/testdata.
package imagefile

import (
	"context"
	"os"
	"testing"
)

func TestWebPStaticAndAnimation(t *testing.T) {
	for _, name := range []string{"static", "animated"} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile("testdata/" + name + ".webp")
			if err != nil {
				t.Fatal(err)
			}
			value, err := Decode(context.Background(), "image.webp", data)
			if err != nil {
				t.Fatal(err)
			}
			if value.Width != 8 || value.Height != 8 || value.Format != "webp" || value.Representative != (name == "animated") {
				t.Fatalf("invalid image metadata: %dx%d %s %v", value.Width, value.Height, value.Format, value.Representative)
			}
			if name == "static" && value.MIME != "image/webp" || name == "animated" && value.MIME != "image/png" {
				t.Fatal("wrong transport format")
			}
		})
	}
}

func FuzzDecode(f *testing.F) {
	for _, name := range []string{"static", "animated"} {
		if data, err := os.ReadFile("testdata/" + name + ".webp"); err == nil {
			f.Add(data)
		}
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1024*1024 {
			return
		}
		_, _ = Decode(context.Background(), "fuzz", data)
	})
}
