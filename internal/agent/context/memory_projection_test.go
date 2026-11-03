// Purpose: Reuse unchanged summaries without retaining stale redactions or large caches.
// Pattern: Context projection behavior tests and workload benchmark.
// Usage: Context tests cover repeated post-compaction model calls.
// Related: internal/agent/context/request.go, internal/agent/context/projection.go.
package context

import (
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func TestSummaryProjectionReuseAndCredentialInvalidation(t *testing.T) {
	var projection Projection
	version, summaryReads := 0, 0
	in := Input{System: "rules", Memory: "summary secret", MaxBytes: 1 << 20, Projection: &projection, RedactionVersion: func() int { return version }}
	in.Redact = func(s string) string {
		if s == in.Memory {
			summaryReads++
		}
		if version > 0 {
			return strings.ReplaceAll(s, "secret", "masked")
		}
		return s
	}
	for range 3 {
		if _, err := Build(in); err != nil {
			t.Fatal(err)
		}
	}
	if summaryReads != 1 {
		t.Fatalf("unchanged summary transformed %d times", summaryReads)
	}
	version++
	request, err := Build(in)
	if err != nil || strings.Contains(request.Messages[1].Content, "secret") {
		t.Fatal("stale protected summary", err)
	}
	in.Memory = "new summary"
	request, err = Build(in)
	if err != nil || !strings.HasSuffix(request.Messages[1].Content, in.Memory) {
		t.Fatal("summary replacement lost", err)
	}
	in.Memory = strings.Repeat("large summary", 25000)
	if _, err := Build(in); err != nil || projection.memory.text != "" {
		t.Fatal("large accepted summary created a retained copy", err)
	}
	in.Memory = "small summary"
	Build(in)
	in.Memory = ""
	if _, err := Build(in); err != nil || projection.memory != (summarySnapshot{}) {
		t.Fatal("removed summary retained its source", err)
	}
}

func BenchmarkSummaryProjection(b *testing.B) {
	var p Projection
	in := Input{Request: modelapi.Request{ModelID: "model"}, System: "rules", Memory: strings.Repeat("work completed\n", 4096), MaxBytes: 16 << 20, Projection: &p, RedactionVersion: func() int { return 0 }, Redact: func(s string) string { return strings.Map(func(r rune) rune { return r }, s) }}
	if _, err := Build(in); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Build(in); err != nil {
			b.Fatal(err)
		}
	}
}
