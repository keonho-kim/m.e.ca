// Purpose: Verify numeric wait arguments match the advertised native schema.
// Pattern: Tool input contract tests.
// Usage: go test ./internal/tools/local/catalog -run TestCommandWait.
// Related: internal/tools/local/catalog/commands.go, internal/agent/tool/json_schema.go.
package catalog

import (
	"encoding/json"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestCommandWaitArgument(t *testing.T) {
	for _, value := range []string{"0", "180", "86400", "-1", "1.5", "86401", "null", "\"180\""} {
		_, _, err := Parse(toolapi.Call{ID: "call", Name: "wait_command", Arguments: json.RawMessage("{\"command_id\":\"cmd_owned\",\"wait_seconds\":" + value + "}")})
		want := value == "0" || value == "180" || value == "86400"
		if (err == nil) != want {
			t.Fatalf("%s: %v", value, err)
		}
	}
	for _, definition := range Definitions() {
		if definition.Name != "wait_command" {
			continue
		}
		properties := definition.JSONSchema()["properties"].(map[string]any)
		field := properties["wait_seconds"].(map[string]any)
		if field["type"] != "integer" || field["minimum"] != 0 || field["maximum"] != 86400 {
			t.Fatal(field)
		}
	}
}
