// Purpose: Verify reusable history projections invalidate on pairing, redaction and source changes.
// Pattern: Context boundary tests and benchmark.
// Usage: go test ./internal/agent/context -run TestProjection -bench BenchmarkProjection.
// Related: internal/agent/context/projection.go, internal/agent/context/request.go.
package context

import (
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestProjectionReuseAndInvalidation(t *testing.T) {
	var p Projection
	version, calls := 0, 0
	in := Input{System: "system", MaxBytes: 16 * 1024 * 1024, Projection: &p, RedactionVersion: func() int { return version }, Redact: func(s string) string {
		calls++
		if version > 0 {
			return strings.ReplaceAll(s, "secret", "masked")
		}
		return s
	}}
	in.Messages = []modelapi.Message{{Role: "assistant", Calls: []toolapi.Call{{ID: "one", Name: "execute_command", Arguments: []byte(`{"command":"pwd"}`)}}}}
	first, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	if first.Messages[1].Role != "user" {
		t.Fatal("unpaired call escaped")
	}
	in.Messages = append(in.Messages, modelapi.Message{Role: "tool", ToolCallID: "one", ToolName: "execute_command", Content: "secret"})
	paired, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	if paired.Messages[1].Role != "assistant" {
		t.Fatal("completed group was not reprojected")
	}
	before := calls
	if _, err := Build(in); err != nil {
		t.Fatal(err)
	}
	if calls-before > 1 {
		t.Fatalf("unchanged history was redacted again: %d", calls-before)
	}
	version++
	got, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	if got.Messages[2].Content != "masked" {
		t.Fatal("stale redaction")
	}
	in.Messages = []modelapi.Message{{Role: "user", Content: "replacement"}}
	got, err = Build(in)
	if err != nil || len(got.Messages) != 2 || got.Messages[1].Content != "replacement" {
		t.Fatalf("reset: %+v %v", got, err)
	}
}

func BenchmarkProjection(b *testing.B) {
	for _, cached := range []bool{false, true} {
		name := "uncached"
		if cached {
			name = "cached"
		}
		b.Run(name, func(b *testing.B) {
			var p Projection
			in := Input{MaxBytes: 16 * 1024 * 1024, Redact: func(s string) string { return strings.Map(func(r rune) rune { return r }, s) }}
			for range 100 {
				in.Messages = append(in.Messages, modelapi.Message{Role: "assistant", Content: strings.Repeat("history ", 2048)})
			}
			if cached {
				in.Projection = &p
				in.RedactionVersion = func() int { return 0 }
			}
			if _, err := Build(in); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := Build(in); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestProjectionFailureAndCredentialChange(t *testing.T) {
	var p Projection
	version := 0
	in := Input{Projection: &p, MaxBytes: 16 * 1024 * 1024, RedactionVersion: func() int { return version }, Redact: func(s string) string { return s }}
	in.Messages = []modelapi.Message{{}}
	for range 2 {
		if _, err := Build(in); err == nil {
			t.Fatal("invalid message was cached after a failed projection")
		}
	}
	in.Messages = []modelapi.Message{{Role: "assistant", Content: "secret"}}
	changed := false
	in.Redact = func(s string) string {
		if s == "secret" && !changed {
			changed = true
			version++
			return s
		}
		return strings.ReplaceAll(s, "secret", "masked")
	}
	got, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	if got.Messages[1].Content != "masked" {
		t.Fatal("credential change escaped request invalidation")
	}
}
