// Purpose: Preserve exact packet JSON while avoiding repeated serialization of encoded records.
// Pattern: Native-encoder reference test and benchmark.
// Usage: Run TestSourcePacket and BenchmarkSourcePacket with -benchmem.
// Related: internal/agent/lat/memory/source_packet.go, internal/agent/lat/memory/tail.go.
package memory

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestSourcePacket(t *testing.T) {
	value, err := json.Marshal(Record{Kind: "outcome", Text: "<tag> & 한글\n"})
	if err != nil {
		t.Fatal(err)
	}
	for _, records := range [][]json.RawMessage{nil, {}, {value}} {
		previous := "summary <tag>\n한글\u2028😀"
		want, err := json.Marshal(struct {
			Memory  string            `json:"independent_review_memory"`
			Records []json.RawMessage `json:"later_original_records_oldest_first"`
		}{previous, records})
		if err != nil {
			t.Fatal(err)
		}
		got, err := sourcePacket("independent_review_memory", "later_original_records_oldest_first", previous, records)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("packet changed: got %s, want %s: %v", got, want, err)
		}
	}
}

func BenchmarkSourcePacket(b *testing.B) {
	records := make([]json.RawMessage, 64)
	for i := range records {
		records[i], _ = json.Marshal(Record{Kind: "outcome", Text: strings.Repeat("Observed work <text> 한글\n", 24)})
	}
	for _, direct := range []bool{false, true} {
		name := "native"
		if direct {
			name = "assembled"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var err error
				if direct {
					_, err = sourcePacket("independent_review_memory", "later_original_records_oldest_first", "memo", records)
				} else {
					_, err = json.Marshal(struct {
						Memory  string            `json:"independent_review_memory"`
						Records []json.RawMessage `json:"later_original_records_oldest_first"`
					}{"memo", records})
				}
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
