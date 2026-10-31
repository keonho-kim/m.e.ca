// Purpose: Verify bounded excerpts authenticate omitted bytes and release streaming decoder input.
// Pattern: Integrity, cancellation and allocation regression tests.
// Usage: go test ./internal/storage/content -run TestExcerpt -bench BenchmarkPayloadExcerpt.
// Related: internal/storage/content/excerpt.go, internal/storage/content/payload.go.
package content

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestExcerptAuthenticatesWholePayload(t *testing.T) {
	raw := []byte(`{"text":"` + strings.Repeat("한국어", 1<<18) + `"}`)
	var c Codec
	defer c.Close()
	compressed, err := c.Encode(raw)
	if err != nil {
		t.Fatal(err)
	}
	var zipped bytes.Buffer
	w := gzip.NewWriter(&zipped)
	_, _ = w.Write(raw)
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	for codec, body := range [][]byte{raw, zipped.Bytes(), compressed} {
		prefix, err := c.Excerpt(context.Background(), codec, len(raw), ID(raw), body, 8192)
		if err != nil || !bytes.Equal(prefix, raw[:8192]) {
			t.Fatal("bad excerpt", codec, err)
		}
		if _, err = c.Excerpt(context.Background(), codec, len(raw), strings.Repeat("0", 64), body, 8192); err == nil {
			t.Fatal("unread tail was not authenticated", codec)
		}
	}
	if full, err := c.Restore(2, len(raw), ID(raw), compressed); err != nil || !bytes.Equal(full, raw) {
		t.Fatal("excerpt corrupted reusable decoder", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = c.Excerpt(ctx, 2, len(raw), ID(raw), compressed, 128); !errors.Is(err, context.Canceled) {
		t.Fatal("cancel ignored", err)
	}
}

func BenchmarkPayloadExcerpt(b *testing.B) {
	raw := []byte(`{"output":"` + strings.Repeat("x", 8<<20) + `"}`)
	var c Codec
	defer c.Close()
	body, err := c.Encode(raw)
	if err != nil {
		b.Fatal(err)
	}
	id := ID(raw)
	for _, full := range []bool{true, false} {
		name := "excerpt"
		if full {
			name = "full"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if full {
					_, err = c.Restore(2, len(raw), id, body)
				} else {
					_, err = c.Excerpt(context.Background(), 2, len(raw), id, body, 8192)
				}
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
