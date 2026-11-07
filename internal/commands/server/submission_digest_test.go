// Purpose: Verify unambiguous normalized intent hashes and measure large input hashing.
// Pattern: Contract test and allocation benchmark.
// Usage: go test ./internal/commands/server -run TestSubmissionDigest.
// Related: internal/commands/server/submission_digest.go.
package server

import (
	"strings"
	"testing"
)

func TestSubmissionDigestIncludesIntentAndIgnoresRequestKey(t *testing.T) {
	base := Submission{Kind: "response", Messages: []InputMessage{{"user", "hello"}}, Stored: true}
	want := submissionDigest(base)
	key := base
	key.Key = "retry"
	if submissionDigest(key) != want {
		t.Fatal("key changed canonical intent")
	}
	for _, change := range []func(*Submission){
		func(v *Submission) { v.Kind = "chat" },
		func(v *Submission) { v.Stored = false },
		func(v *Submission) { v.Background = true },
		func(v *Submission) { v.Previous = "previous" },
		func(v *Submission) { v.Messages = []InputMessage{{"assistant", "hello"}} },
		func(v *Submission) { v.Messages = []InputMessage{{"user", "hell"}, {"o", ""}} },
		func(v *Submission) { v.Messages = []InputMessage{{"user", "hello\x00"}} },
	} {
		next := base
		change(&next)
		if submissionDigest(next) == want {
			t.Fatal("distinct intent reused a digest", next)
		}
	}
}

func BenchmarkSubmissionDigest(b *testing.B) {
	for _, character := range []string{"x", "<"} {
		b.Run(character, func(b *testing.B) {
			input := Submission{Kind: "response", Stored: true, Messages: []InputMessage{{"user", strings.Repeat(character, 1<<20)}}}
			b.SetBytes(1 << 20)
			b.ReportAllocs()
			for b.Loop() {
				submissionDigest(input)
			}
		})
	}
}
