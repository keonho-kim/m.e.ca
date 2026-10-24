// Purpose: Verify the advertised tool set and rejection of retired move calls without side effects.
// Pattern: Tool boundary regression tests.
// Usage: go test ./internal/tools/local -run TestCatalog.
// Related: internal/tools/local/catalog.go, internal/tools/local/arguments.go, internal/tools/local/prepare.go.
package local

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

func TestCatalogUsesShellForMoves(t *testing.T) {
	e, project := testExecutor(t)
	var names []string
	for _, definition := range e.Definitions() {
		names = append(names, definition.Name)
	}
	slices.Sort(names)
	want := []string{"create_directory", "edit_file", "execute_command", "write_file"}
	if !slices.Equal(names, want) {
		t.Fatalf("advertised tools = %v; want %v", names, want)
	}
	source := filepath.Join(project, "source")
	if err := os.WriteFile(source, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("move_file", `{"path":"source","destination":"destination"}`))
	if err == nil || !strings.Contains(err.Error(), "unknown tool") {
		t.Fatalf("retired move call was not rejected: %v", err)
	}
	if body, err := os.ReadFile(source); err != nil || string(body) != "keep" {
		t.Fatalf("source changed: %q %v", body, err)
	}
	if _, err := os.Lstat(filepath.Join(project, "destination")); !os.IsNotExist(err) {
		t.Fatalf("retired call created a destination: %v", err)
	}
}

func TestCatalogDefinitionsDoNotExposePrivateSchema(t *testing.T) {
	e := &Executor{}
	before, err := json.Marshal(e.Definitions())
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
	for _, definition := range e.Definitions() {
		mutate(definition.Parameters)
	}
	after, err := json.Marshal(e.Definitions())
	if err != nil || string(before) != string(after) {
		t.Fatal("returned definitions exposed shared mutable schemas")
	}
	if _, _, err := parseArguments(call("edit_file", `{"path":"a","edits":[{"old_text":"old","new_text":"new"}],"format_policy":"preserve"}`)); err != nil {
		t.Fatal("a caller changed private argument validation:", err)
	}
}
