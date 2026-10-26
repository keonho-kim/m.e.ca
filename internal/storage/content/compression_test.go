// Purpose: Verify codec frame bounds, concurrent use and explicit workspace cleanup.
// Pattern: Content codec contract tests.
// Usage: go test -race ./internal/storage/content -run TestPayloadCodec.
// Related: internal/storage/content/compression.go.
package content

import (
	"bytes"
	"strings"
	"sync"
	"testing"
)

func TestPayloadCodecBoundsAndCleanup(t *testing.T) {
	var codec Codec
	raw := []byte(strings.Repeat("code and reasoning\n", 10000))
	compressed, err := codec.Encode(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := codec.decode(compressed, len(raw))
	if err != nil || !bytes.Equal(got, raw) {
		t.Fatal("round trip failed", err)
	}
	for _, size := range []int{0, len(raw) - 1, len(raw) + 1, MaxBytes + 1} {
		if _, err := codec.decode(compressed, size); err == nil {
			t.Fatalf("accepted incorrect size %d", size)
		}
	}
	if _, err := codec.decode(compressed[:len(compressed)-1], len(raw)); err == nil {
		t.Fatal("truncated payload accepted")
	}
	if err := codec.Close(); err != nil || codec.encoder != nil || codec.decoder != nil || codec.buffer != nil {
		t.Fatal("codec resources retained after close", err)
	}
	if _, err := codec.Encode(raw); err == nil {
		t.Fatal("closed codec accepted a write")
	}
	if _, err := codec.decode(compressed, len(raw)); err == nil {
		t.Fatal("closed codec accepted a read")
	}
	if err := codec.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPayloadCodecConcurrentUse(t *testing.T) {
	var codec Codec
	t.Cleanup(func() { codec.Close() })
	var workers sync.WaitGroup
	for i := range 8 {
		workers.Go(func() {
			raw := []byte(strings.Repeat(string(rune('a'+i)), 50000))
			compressed, err := codec.Encode(raw)
			if err != nil {
				t.Error(err)
				return
			}
			got, err := codec.decode(compressed, len(raw))
			if err != nil || !bytes.Equal(got, raw) {
				t.Error("concurrent round trip failed", err)
			}
		})
	}
	workers.Wait()
}
