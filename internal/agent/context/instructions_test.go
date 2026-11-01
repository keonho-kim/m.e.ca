// Purpose: Verify fixed system projection reuse remains bound to source, root and secret revision.
// Pattern: Redaction boundary regression tests.
// Usage: Run TestFixedInstructions with the context package tests.
// Related: internal/agent/context/instructions.go, internal/agent/context/projection.go.
package context

import (
	"strings"
	"testing"
)

func TestFixedInstructionsReuseAndInvalidation(t *testing.T) {
	var p Projection
	version, calls := 0, 0
	input := Input{System: "system secret", ProjectRoot: "/project", Projection: &p, MaxBytes: 1 << 20, RedactionVersion: func() int { return version }, Redact: func(s string) string {
		calls++
		if version > 0 {
			return strings.ReplaceAll(s, "secret", "masked")
		}
		return s
	}}
	first, err := Build(input)
	if err != nil {
		t.Fatal(err)
	}
	before := calls
	if _, err := Build(input); err != nil || calls != before {
		t.Fatal("unchanged system was redacted again", err, calls, before)
	}
	version++
	request, err := Build(input)
	if err != nil || strings.Contains(request.Messages[0].Content, "secret") {
		t.Fatal("stale system redaction", err)
	}
	input.ProjectRoot = "/another-project"
	request, err = Build(input)
	if err != nil || !strings.Contains(request.Messages[0].Content, input.ProjectRoot) {
		t.Fatal("stale project root", err)
	}
	if !strings.Contains(first.Messages[0].Content, "/project") {
		t.Fatal("earlier system projection mutated")
	}
	input.System = strings.Repeat("x", 256*1024+1)
	if _, err := Build(input); err != nil || p.system.text != "" {
		t.Fatal("large accepted instructions created an unbounded cache", err)
	}
}

func TestFixedInstructionsRetryWhenSecretChangesDuringProjection(t *testing.T) {
	var p Projection
	version := 0
	input := Input{System: "system secret", Projection: &p, MaxBytes: 8192, RedactionVersion: func() int { return version }}
	input.Redact = func(s string) string {
		if version == 0 {
			version++
			return s
		}
		return strings.ReplaceAll(s, "secret", "masked")
	}
	request, err := Build(input)
	if err != nil || strings.Contains(request.Messages[0].Content, "secret") {
		t.Fatal("revision barrier did not cover fixed instructions", err)
	}
}
