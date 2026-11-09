// Purpose: Measure repeated nested container reconstruction for image-like payload arrays.
// Pattern: Persistence workload benchmark.
// Usage: go test ./internal/storage/content -run ^$ -bench BenchmarkRestoreArray -benchmem.
// Related: internal/storage/content/restore.go.
package content

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func BenchmarkRestoreArray(b *testing.B) {
	for _, count := range []int{1, 4, 16, 32} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			images := make([]map[string]string, count)
			for i := range images {
				images[i] = map[string]string{"data": strings.Repeat("x", 64<<10) + fmt.Sprint(i)}
			}
			body, _ := json.Marshal(map[string]any{"role": "user", "content": "Inspect images", "images": images})
			values := map[string]json.RawMessage{}
			packed, refs, err := Pack(body, "", func(raw []byte) (string, error) {
				id := ID(raw)
				values[id] = append(json.RawMessage(nil), raw...)
				return id, nil
			})
			if err != nil {
				b.Fatal(err)
			}
			encoded, _ := json.Marshal(refs)
			b.ReportAllocs()
			b.SetBytes(int64(len(body)))
			b.ResetTimer()
			for b.Loop() {
				if _, err := Restore(packed, encoded, nil, func(id string) (json.RawMessage, error) { return values[id], nil }); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
