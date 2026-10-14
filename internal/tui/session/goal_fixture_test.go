// Purpose: Answer hidden Goal analysis requests in conversation workflow fixtures.
// Pattern: HTTP test fixture.
// Usage: Session context, compaction and streaming integration tests call respondGoalAnalysis.
// Related: internal/tui/session/context_test.go, internal/tui/session/compaction_test.go, internal/tui/session/stream_test.go.
package session

import (
	"encoding/json"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"net/http"
	"strings"
	"testing"
)

func respondGoalAnalysis(t *testing.T, w http.ResponseWriter, messages []modelapi.Message) bool {
	t.Helper()
	if len(messages) == 0 || !strings.Contains(messages[len(messages)-1].Content, "\nGOAL ANALYSIS:") {
		return false
	}
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(map[string]any{
		"choices": []any{map[string]any{"message": map[string]string{"content": "GOAL: Respond to the current request\nTASK answer: Respond to the current request"}, "finish_reason": "stop"}},
		"usage":   map[string]int{"prompt_tokens": 20, "completion_tokens": 7},
	})
	if err != nil {
		t.Error(err)
	}
	return true
}
