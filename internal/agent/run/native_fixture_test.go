// Purpose: Build complete native tool responses for deterministic runtime tests.
// Pattern: Test fixture.
// Usage: Scripted generators emit these calls as provider events, never as assistant text.
// Related: internal/agent/run/run_test.go, internal/agent/model/model.go.
package run

import (
	"encoding/json"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func nativeCall(name string, arguments map[string]any) string {
	if arguments == nil {
		arguments = map[string]any{}
	}
	body, err := json.Marshal(arguments)
	if err != nil {
		panic(err)
	}
	encoded, err := json.Marshal(toolapi.Call{ID: "assigned-by-generator", Name: name, Arguments: body})
	if err != nil {
		panic(err)
	}
	return "native:" + string(encoded)
}
