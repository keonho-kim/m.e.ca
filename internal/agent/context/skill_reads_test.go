// Purpose: Verify current-request skill reads survive compaction without duplicate context copies.
// Pattern: Context contract tests.
// Usage: go test ./internal/agent/context -run TestSkillReads.
// Related: internal/agent/context/skill_reads.go, internal/agent/context/request.go.
package context

import (
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func TestSkillReadsPreserveExactInstructionsOnce(t *testing.T) {
	read := modelapi.Message{ID: "skill_note", Role: "user", Content: "SKILL unique-instruction secret"}
	for _, history := range [][]modelapi.Message{{read}, nil} {
		request, err := Build(Input{System: "Policy", Messages: history, SkillReads: []modelapi.Message{read}, Redact: func(s string) string { return strings.ReplaceAll(s, "secret", "[redacted]") }, MaxBytes: 10000})
		if err != nil {
			t.Fatal(err)
		}
		text := ""
		for _, m := range request.Messages {
			text += m.Content
		}
		if strings.Count(text, "unique-instruction") != 1 || strings.Contains(text, "secret") {
			t.Fatalf("context=%q", text)
		}
	}
}
