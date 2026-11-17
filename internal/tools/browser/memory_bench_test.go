// Purpose: Measure capture, projection and page allocations separately for representative browser observations.
// Pattern: Bounded reproducible benchmarks without browser or network startup.
// Usage: go test ./internal/tools/browser -run '^$' -bench BenchmarkBrowserMemory -benchmem.
// Related: internal/tools/browser/capture.go, internal/tools/browser/results.go.
package browser

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func BenchmarkBrowserMemory(b *testing.B) {
	for _, size := range []int{32 << 10, 3 << 20} {
		b.Run(fmt.Sprintf("%dKiB", size>>10), func(b *testing.B) {
			for _, unicode := range []bool{false, true} {
				name, unit := "ascii", "x"
				if unicode {
					name, unit = "escaped-unicode", "界😀\n\t"
				}
				text := strings.Repeat(unit, size/len(unit))
				raw, _ := json.Marshal(map[string]string{"text": text})
				envelope, _ := json.Marshal(struct {
					Success bool            `json:"success"`
					Data    json.RawMessage `json:"data"`
				}{true, raw})
				b.Run(name, func(b *testing.B) {
					for _, unchanged := range []bool{false, true} {
						mode := "distinct"
						if unchanged {
							mode = "unchanged"
						}
						b.Run(mode, func(b *testing.B) {
							s := New("", "", func(text string) string { return text }, nil)
							s.work = b.TempDir()
							s.setAddress("http://fixture/initial")
							if _, err := s.store(raw, 0); err != nil {
								b.Fatal(err)
							}
							b.ReportAllocs()
							b.SetBytes(int64(len(envelope)))
							b.ResetTimer()
							for i := 0; i < b.N; i++ {
								if !unchanged {
									s.setAddress(fmt.Sprintf("http://fixture/%d", i))
								}
								capture := &boundedBuffer{remaining: maxCapture}
								for offset := 0; offset < len(envelope); offset += 32 << 10 {
									end := min(offset+32<<10, len(envelope))
									if _, err := capture.Write(envelope[offset:end]); err != nil {
										b.Fatal(err)
									}
								}
								response, err := parseResponse(capture.Bytes())
								if err != nil {
									b.Fatal(err)
								}
								page, err := s.store(response.Data, 0)
								if err != nil {
									b.Fatal(err)
								}
								if len(page.Text) > 8192 {
									b.Fatal("unbounded presentation")
								}
							}
						})
					}
				})
				b.Run(name+"/page", func(b *testing.B) {
					s := New("", "", func(text string) string { return text }, nil)
					s.work = b.TempDir()
					first, err := s.store(raw, 0)
					if err != nil {
						b.Fatal(err)
					}
					r := Request{ResultID: first.ResultID, Revision: first.Revision, Limit: 8192}
					b.ReportAllocs()
					b.ResetTimer()
					for b.Loop() {
						if _, err := s.page(r); err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		})
	}
}
