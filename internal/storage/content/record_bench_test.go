// Purpose: Measure large record packing and selective restoration without database latency.
// Pattern: Persistence workload benchmark.
// Usage: Run BenchmarkRecordContent with -benchmem.
// Related: internal/storage/content/pack.go, internal/storage/content/restore.go.
package content

import (
	"encoding/json"
	"strings"
	"testing"
)

func BenchmarkRecordContent(b *testing.B) {
	body, err := json.Marshal(map[string]any{
		"role": "assistant", "content": strings.Repeat("source code & 한국어 <module>\n", 4096),
		"calls": []any{map[string]any{"id": "call_1", "arguments": map[string]string{"path": "server.go", "content": strings.Repeat("code\n", 8192)}}},
	})
	if err != nil {
		b.Fatal(err)
	}
	values := map[string]json.RawMessage{}
	put := func(raw []byte) (string, error) {
		id := ID(raw)
		values[id] = append(json.RawMessage(nil), raw...)
		return id, nil
	}
	packed, refs, err := Pack(body, put)
	if err != nil {
		b.Fatal(err)
	}
	b.Run("pack", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(body)))
		for b.Loop() {
			if _, _, err := Pack(body, put); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("restore", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(body)))
		for b.Loop() {
			if _, err := Restore(packed, refs, nil, func(id string) (json.RawMessage, error) { return values[id], nil }); err != nil {
				b.Fatal(err)
			}
		}
	})
}
