// Purpose: Prevent a wide Zstandard history window from amplifying a small preview's memory use.
// Pattern: Valid-frame allocation regression test at the stored-format boundary.
// Usage: go test ./internal/storage/content -run TestPrefixWideWindow.
// Related: internal/storage/content/prefix_reader.go, internal/storage/content/compression.go.
package content

import (
	"runtime"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func TestPrefixWideWindowDoesNotAllocateItsDeclaredHistory(t *testing.T) {
	raw := []byte(`"` + strings.Repeat("x", 64<<10) + `"`)
	encoder, err := zstd.NewWriter(nil, zstd.WithWindowSize(8<<20), zstd.WithSingleSegment(false), zstd.WithEncoderConcurrency(1))
	if err != nil {
		t.Fatal(err)
	}
	compressed := encoder.EncodeAll(raw, nil)
	if err := encoder.Close(); err != nil {
		t.Fatal(err)
	}
	// A non-single-segment frame may declare a wider window than its body
	// requires. The descriptor is byte five; log2(8MiB)-10 is its exponent.
	if compressed[4]&0x20 != 0 {
		t.Fatal("fixture unexpectedly used a single segment")
	}
	compressed[5] = 13 << 3
	var header zstd.Header
	if err := header.Decode(compressed); err != nil || header.WindowSize < 8<<20 {
		t.Fatal("fixture did not declare a wide window", header, err)
	}
	var codec Codec
	defer codec.Close()
	if _, err := codec.Restore(2, len(raw), ID(raw), compressed); err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	text, clipped, err := codec.TextPrefix(t.Context(), 2, len(raw), ID(raw), compressed, 16)
	runtime.ReadMemStats(&after)
	if err != nil || text != strings.Repeat("x", 16) || !clipped {
		t.Fatal("valid wide-window text changed", text, clipped, err)
	}
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 2<<20 {
		t.Fatalf("64KiB original with an 8MiB window allocated %d bytes", allocated)
	}
}
