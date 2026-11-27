// Purpose: Encode code-mode calls for Chat Completions integration fixtures.
// Pattern: Test-only wire adapter over the shared host-call fixture builder.
// Usage: CLI and TUI HTTP fixtures return these calls from their local model servers.
// Related: internal/agent/modeltest/codemode.go, cmd/meca/headless_test.go, internal/tui/session/readme_test.go.
package modeltest

import "github.com/keonho-kim/meca/internal/agent/tool"

func ChatCalls(calls []tool.Call) []map[string]any {
	code := Calls(calls)
	encoded := make([]map[string]any, len(code))
	for i, call := range code {
		encoded[i] = map[string]any{
			"id": call.ID, "type": "function",
			"function": map[string]string{"name": call.Name, "arguments": string(call.Arguments)},
		}
	}
	return encoded
}
