// Purpose: Verify advertised tools and reject unsupported calls without changing files.
// Pattern: Tool execution boundary tests.
// Usage: go test ./internal/tools/local -run TestCatalog.
// Related: internal/tools/local/catalog/catalog.go, internal/tools/local/prepare.go.
package local

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/policy"
)

func TestCatalogUsesShellForMoves(t *testing.T) {
	e, project := testExecutor(t)
	var names []string
	for _, definition := range e.Definitions() {
		names = append(names, definition.Name)
	}
	slices.Sort(names)
	want := []string{"cancel_command", "create_directory", "edit_file", "execute_command", "list_command_runs", "wait_command", "write_file"}
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
