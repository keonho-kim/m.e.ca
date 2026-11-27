// Purpose: Build code-based tool responses for deterministic runtime tests.
// Pattern: Test fixture.
// Usage: Scripted generators emit these calls as provider events, never as assistant text.
// Related: internal/agent/run/run_test.go, internal/agent/model/model.go.
package run

import (
	"encoding/json"
	"github.com/keonho-kim/meca/internal/agent/modeltest"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func nativeCall(name string, arguments map[string]any) string {
	if arguments == nil {
		arguments = map[string]any{}
	}
	body, err := json.Marshal(arguments)
	if err != nil {
		panic(err)
	}
	call := toolapi.Call{ID: "assigned-by-generator", Name: name, Arguments: body}
	encoded, err := json.Marshal(modeltest.Calls([]toolapi.Call{call})[0])
	if err != nil {
		panic(err)
	}
	return "native:" + string(encoded)
}

func codeOutput(body, tool string) json.RawMessage {
	var result struct {
		Result       json.RawMessage   `json:"result"`
		Observations []codeObservation `json:"observations"`
	}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		panic(err)
	}
	for _, observation := range result.Observations {
		if observation.Tool == tool {
			return observation.Value
		}
	}
	return result.Result
}

func innerOutcomes(j *journal) []runrecord.Event {
	var outcomes []runrecord.Event
	for _, event := range j.events {
		if event.Call != nil && event.Call.ParentExecutionID != "" && (event.Kind == "tool_succeeded" || event.Kind == "tool_failed") {
			outcomes = append(outcomes, event)
		}
	}
	return outcomes
}

func callText(call toolapi.Call, field string) string {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(call.Arguments, &values); err != nil {
		panic(err)
	}
	var value string
	if err := json.Unmarshal(values[field], &value); err != nil {
		panic(err)
	}
	return value
}
