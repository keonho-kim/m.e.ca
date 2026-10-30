// Purpose: Preserve bounded snapshot reads across growth, truncation, cancellation and partial I/O.
// Pattern: Read boundary regression tests and allocation benchmark.
// Usage: Local image tests exercise source changes without racing real filesystem timing.
// Related: internal/tools/local/image_read.go, internal/tools/local/image.go.
package local

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/media"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type imageReadFunc func([]byte) (int, error)

func (f imageReadFunc) Read(p []byte) (int, error) { return f(p) }

func TestReadImageBytes(t *testing.T) {
	for _, tc := range []struct {
		name string
		size int64
	}{
		{"unchanged", 4}, {"grew", 3}, {"truncated", 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := readImageBytes(context.Background(), bytes.NewBufferString("data"), tc.size)
			if tc.size == 4 {
				if err != nil || string(got) != "data" {
					t.Fatalf("snapshot=%q err=%v", got, err)
				}
				return
			}
			var stale *toolapi.Failure
			if !errors.As(err, &stale) || stale.Code != toolapi.StaleTarget {
				t.Fatalf("source change did not produce a stale-target error: %v", err)
			}
		})
	}
	for _, size := range []int64{-1, media.MaxBytes + 1} {
		if _, err := readImageBytes(context.Background(), imageReadFunc(func([]byte) (int, error) {
			t.Fatal("invalid size read the source")
			return 0, io.EOF
		}), size); err == nil {
			t.Fatal("invalid size accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	reader := imageReadFunc(func(p []byte) (int, error) {
		p[0] = 'x'
		cancel()
		return 1, nil
	})
	if _, err := readImageBytes(ctx, reader, 4); !errors.Is(err, context.Canceled) {
		t.Fatalf("mid-read cancellation lost: %v", err)
	}
	failure := errors.New("source I/O failed")
	if _, err := readImageBytes(context.Background(), imageReadFunc(func([]byte) (int, error) { return 0, failure }), 4); !errors.Is(err, failure) {
		t.Fatalf("source error lost: %v", err)
	}
}

func BenchmarkReadImageBytes(b *testing.B) {
	data := make([]byte, 8*1024*1024)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := readImageBytes(context.Background(), bytes.NewReader(data), int64(len(data))); err != nil {
			b.Fatal(err)
		}
	}
}
