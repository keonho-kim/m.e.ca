// Purpose: Measure phase input preparation with large reusable requirements and evidence.
// Pattern: Fixed-input allocation benchmark.
// Usage: Run BenchmarkPhasePacket with -benchmem.
// Related: internal/agent/lat/packet.go, internal/agent/run/lat_fork.go.
package lat

import (
	"encoding/json"
	"strings"
	"testing"
)

func BenchmarkPhasePacket(b *testing.B) {
	input := Input{Prompt: "Review the proposed plan.", Instructions: strings.Repeat("Shared rule.\n", 1200), Objective: strings.Repeat("요구사항 ", 400), Plan: strings.Repeat("Plan step.\n", 400), Critique: "Keep the current scope.", Context: json.RawMessage(`{"evidence":"` + strings.Repeat("original observation ", 1200) + `"}`), ContextWindow: 262144}
	b.ReportAllocs()
	for b.Loop() {
		frame, err := Prepare(input)
		if err != nil {
			b.Fatal(err)
		}
		if len(input.Context) > frame.SourceBudget() {
			b.Fatal("evidence exceeds budget")
		}
		if _, err := frame.Build(input.Context); err != nil {
			b.Fatal(err)
		}
	}
}
