//go:build !tui

// Purpose: Drive real team work declarations and final reporting without credentials or paid inference.
// Pattern: Deterministic native-tool provider fixture.
// Usage: The browser fixture selects this script for TEAM_PRESENTATION_FIXTURE requests.
// Related: cmd/meca/ui_provider_test.go, webui/e2e/team-presentation.spec.ts.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type uiFixtureMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

func stringMustMessages(messages []uiFixtureMessage) string {
	var text strings.Builder
	for _, m := range messages {
		text.Write(m.Content)
	}
	return text.String()
}
func uiTeamPresentationReply(w http.ResponseWriter, messages []uiFixtureMessage) {
	w.Header().Set("Content-Type", "text/event-stream")
	work, report := false, false
	for _, m := range messages {
		if m.Role == "tool" {
			work = work || strings.Contains(string(m.Content), "work_revision")
			report = report || strings.Contains(string(m.Content), "report_required")
		}
	}
	delta := map[string]any{}
	reason := "tool_calls"
	name, arguments := "set_work", `{"summary":"TEAM_WORK_PROSE_CANARY: I own API verification.","paths":[],"contract":"TEAM_CONTRACT_CANARY: Preserve typed completion statistics."}`
	if work {
		name, arguments = "leave_team", `{"message":"TEAM_PUBLIC_MESSAGE_CANARY: Contribution ready."}`
	}
	if report {
		delta["content"] = "TEAM_FINAL_RESPONSE_CANARY: All checks finished."
		reason = "stop"
	} else {
		delta["tool_calls"] = []any{map[string]any{"index": 0, "id": "fixture-" + name, "type": "function", "function": map[string]string{"name": name, "arguments": arguments}}}
	}
	data, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"delta": delta, "finish_reason": reason}}})
	fmt.Fprintf(w, "data: %s\n\n", data)
	fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":32,\"completion_tokens\":16}}\n\ndata: [DONE]\n\n")
	_ = http.NewResponseController(w).Flush()
}
