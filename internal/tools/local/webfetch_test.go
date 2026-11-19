// Purpose: Check that WebFetch is hidden and unavailable when its runtime is unavailable.
// Pattern: Catalog and execution admission regression.
// Usage: go test ./internal/tools/local -run TestWebFetchUnavailableWithoutDocumentReader.
// Related: internal/tools/local/catalog.go, internal/tools/local/webfetch.go.
package local

import (
	"os"
	"path/filepath"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tools/browser/webfetch"
)

func TestWebFetchAcceptsPublicURLWithoutRegistration(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MECA_BROWSER_EXECUTABLE", executable)
	client := webfetch.NewClient()
	home := t.TempDir()
	e, err := NewExecutor(t.TempDir(), home, filepath.Join(home, ".meca"), func(value string) string { return value }, Options{WebFetch: client})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for _, definition := range e.Definitions() {
		if definition.Name == toolapi.WebFetch {
			op, err := e.Prepare(t.Context(), policy.Settings{Scope: policy.System}, call(toolapi.WebFetch, `{"url":"https://example.org/docs"}`))
			if op == nil || err != nil || op.Approval().Required {
				t.Fatalf("public URL required additional setup: %v", err)
			}
			return
		}
	}
	t.Fatal("WebFetch was omitted without an address registration")
}

func TestWebFetchRequiresInstalledChrome(t *testing.T) {
	t.Setenv("MECA_BROWSER_EXECUTABLE", filepath.Join(t.TempDir(), "missing-chrome"))
	client := webfetch.NewClient()
	home := t.TempDir()
	e, err := NewExecutor(t.TempDir(), home, filepath.Join(home, ".meca"), func(value string) string { return value }, Options{WebFetch: client})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for _, definition := range e.Definitions() {
		if definition.Name == toolapi.WebFetch {
			t.Fatal("advertised WebFetch without an installed Chrome")
		}
	}
	op, err := e.Prepare(t.Context(), policy.Settings{Scope: policy.System}, call(toolapi.WebFetch, `{"url":"http://127.0.0.1:8080"}`))
	if op != nil || err == nil {
		t.Fatal("prepared WebFetch without an installed Chrome")
	}
}

func TestWebFetchUnavailableWithoutDocumentReader(t *testing.T) {
	e, _ := testExecutor(t)
	for _, definition := range e.Definitions() {
		if definition.Name == toolapi.WebFetch {
			t.Fatal("advertised WebFetch without a document reader")
		}
	}
	operation, err := e.Prepare(t.Context(), policy.Settings{Scope: policy.System, Permission: policy.PermissionAllMighty}, call(toolapi.WebFetch, `{"url":"http://127.0.0.1:8080"}`))
	if operation != nil || err == nil {
		t.Fatal("unavailable WebFetch was prepared under All Mighty")
	}
}
