// Purpose: Verify supported tool names and independent native argument-validation schemas.
// Pattern: Registry ownership contract tests.
// Usage: go test ./internal/tools/local/catalog.
// Related: internal/tools/local/catalog/catalog.go, internal/tools/local/catalog/arguments.go.
package catalog

import (
	"encoding/json"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"slices"
	"testing"
)

func TestCatalogDefinitionsDoNotExposePrivateSchema(t *testing.T) {

	before, err := json.Marshal(Definitions())
	if err != nil {
		t.Fatal(err)
	}
	var mutate func([]toolapi.Parameter)
	mutate = func(parameters []toolapi.Parameter) {
		for i := range parameters {
			parameters[i].Name = "modified"
			for j := range parameters[i].Values {
				parameters[i].Values[j] = "modified"
			}
			mutate(parameters[i].Fields)
		}
	}
	for _, definition := range Definitions() {
		mutate(definition.Parameters)
	}
	after, err := json.Marshal(Definitions())
	if err != nil || string(before) != string(after) {
		t.Fatal("returned definitions exposed shared mutable schemas")
	}
	if _, _, err := Parse(toolapi.Call{ID: "call_catalog", Name: "edit_file", Arguments: json.RawMessage(`{"path":"a","edits":[{"old_text":"old","new_text":"new"}],"format_policy":"preserve"}`)}); err != nil {
		t.Fatal("a caller changed private argument validation:", err)
	}
}

func TestCatalogAdvertisesSupportedTools(t *testing.T) {
	var names []string
	for _, definition := range Definitions() {
		names = append(names, definition.Name)
	}
	slices.Sort(names)
	want := []string{"cancel_command", "create_directory", "edit_file", "execute_command", "list_command_runs", "read_image", "wait_command", "write_file"}
	if !slices.Equal(names, want) {
		t.Fatalf("advertised tools = %v; want %v", names, want)
	}
}
