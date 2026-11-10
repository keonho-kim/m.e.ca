// Purpose: Compare supported full decoding and bounded preview decoding for a wide frame.
// Pattern: Compression format regression test.
// Usage: Run go test for this package; failures retain task and storage ownership.
// Related: internal/storage/content/prefix_stream.go.
package content

import (
	"bytes"
	"github.com/klauspost/compress/zstd"
	"strings"
	"testing"
)

func TestWideFramePreview(t *testing.T) {
	raw := []byte(`"` + strings.Repeat("abcdef", 50000) + `"`)
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1), zstd.WithWindowSize(1<<20))
	if err != nil {
		t.Fatal(err)
	}
	body := encoder.EncodeAll(raw, nil)
	encoder.Close()
	var codec Codec
	defer codec.Close()
	if _, err := codec.Restore(2, len(raw), ID(raw), body); err != nil {
		t.Fatal("fixture was not a supported payload", err)
	}
	if _, _, err := StreamTextPrefix(t.Context(), 2, len(raw), ID(raw), bytes.NewReader(body), 128, false, nil); err != nil {
		t.Fatalf("valid payload cannot be previewed: %v", err)
	}
}
