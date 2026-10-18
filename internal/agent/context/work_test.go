// Purpose: Keep follow-up requirements and stated decisions separate from observations and private reasoning.
// Pattern: Pure context contract tests.
// Usage: go test ./internal/agent/context -run TestWork.
// Related: internal/agent/context/work.go.
package context

import (
	"encoding/json"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"testing"
)

func TestWorkKeepsFollowupAndVisibleApproach(t *testing.T) {
	current := "이제 기본 구현을 해줘"
	messages := []modelapi.Message{
		{Role: "user", Content: "Create the FastAPI project with empty files"},
		{Role: "assistant", Content: "Files created"},
		{Role: "user", Content: current},
		{Role: "assistant", Content: "Implement users and posts with SQLite first.", Reasoning: "PRIVATE_THOUGHT", Calls: []toolapi.Call{{Name: "inspect_fixture"}}},
		{Role: "tool", Content: "PRIVATE_TOOL_DUMP"},
		{Role: "assistant", Content: "Implement the database connection next.", Calls: []toolapi.Call{{Name: "write_file"}}},
	}
	b := Work(current, messages, "Earlier settings remain local.")
	if len(b.EarlierRequests) != 1 || len(b.ProposedApproach) != 2 || b.EarlierWork == "" {
		t.Fatalf("%+v", b)
	}
	body, _ := json.Marshal(b)
	if strings.Contains(string(body), "PRIVATE_") || strings.Contains(string(body), current) {
		t.Fatal("duplicated current request or unrelated internals")
	}
}

func TestWorkKeepsNewApproachAfterRequestWasCompacted(t *testing.T) {
	b := Work("Implement the service", []modelapi.Message{
		{Role: "assistant", Content: "Fix the observed import error next.", Calls: []toolapi.Call{{Name: "edit_file"}}},
		{Role: "tool", Content: "File updated"},
	}, "Earlier project setup")
	if len(b.ProposedApproach) != 1 || len(b.EarlierRequests) != 0 {
		t.Fatalf("lost the recent action after compaction: %+v", b)
	}
}
