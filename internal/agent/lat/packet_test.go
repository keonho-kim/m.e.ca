// Purpose: Preserve current requirements and isolated evidence within final review budgets.
// Pattern: Context boundary regression tests.
// Usage: go test ./internal/agent/lat.
// Related: internal/agent/lat/packet.go, internal/agent/lat/memory/tail.go.
package lat

import (
	"encoding/json"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	"strings"
	"testing"
)

func TestPacketPreservesIndependentEvidence(t *testing.T) {
	input := Input{Prompt: prompt.Defaults().LAT, Instructions: "PROJECT_RULE", Objective: "이제 구현해줘", Context: json.RawMessage(`{"independent_review_memory":"Create a simple API","later_original_records_oldest_first":[{"kind":"outcome","text":"actual result"}]}`), Draft: strings.Repeat("작업 결과 ", 10000), ContextWindow: 8192, MaxOutput: 4096}
	request, err := Build(input)
	if err != nil {
		t.Fatal(err)
	}
	if request.MaxOutput != 2048 || len(request.Tools) != 0 || len(request.Messages) != 2 {
		t.Fatal("review capabilities or budget changed")
	}
	encoded, _ := json.Marshal(request.Messages)
	for _, value := range []string{"PROJECT_RULE", "이제 구현해줘", "Create a simple API", "actual result"} {
		if !strings.Contains(string(encoded), value) {
			t.Fatal("missing requirement/evidence", value)
		}
	}
	size := 1024 + request.MaxOutput
	for _, message := range request.Messages {
		size += len(message.Content)
	}
	if size > 8192 {
		t.Fatal("exceeded context", size)
	}
}

func TestPacketNeverClipsCurrentUserRequirement(t *testing.T) {
	objective := strings.Repeat("requirement ", 1400) + "MANDATORY_END"
	request, err := Build(Input{Prompt: prompt.Defaults().LAT, Objective: objective, Context: json.RawMessage(`{}`), Draft: "done", ContextWindow: 65536})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(request.Messages[1].Content, "MANDATORY_END") {
		t.Fatal("current user requirement truncated")
	}
	_, err = Build(Input{Prompt: prompt.Defaults().LAT, Objective: objective, Context: json.RawMessage(`{}`), ContextWindow: 8192})
	if err == nil {
		t.Fatal("unfittable mandatory request silently clipped")
	}
}

func TestPacketRejectsUnfittableIndependentEvidence(t *testing.T) {
	_, err := Build(Input{Prompt: prompt.Defaults().LAT, Objective: "work", Context: json.RawMessage(`{"text":"` + strings.Repeat("x", 100000) + `"}`), ContextWindow: 8192})
	if err == nil {
		t.Fatal("oversized evidence silently omitted")
	}
	_, err = Build(Input{Prompt: prompt.Defaults().LAT, Instructions: strings.Repeat("rule", 4096), Context: json.RawMessage(`{}`), ContextWindow: 8192})
	if err == nil {
		t.Fatal("unfittable instructions accepted")
	}
}
