// Purpose: Return the registered workspace identity without asking clients to compare path spellings.
// Pattern: Native application contract over a real canonical workspace registry.
// Usage: go test ./internal/serverapi -run TestWorkspaceRegistrationIdentity.
// Related: internal/serverapi/application.go, api/meca/v1/application.proto.
package serverapi

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/commands/server"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"github.com/keonho-kim/meca/internal/tools/local/filesystem"
)

func TestWorkspaceRegistrationIdentity(t *testing.T) {
	f := newFixture(t, nil, false)
	saves := 0
	workspaces, err := server.NewWorkspaces(t.Context(), server.WorkspaceOptions{
		Default: f.project, Check: filesystem.Canonical,
		Save: func(context.Context, string) error { saves++; return nil },
		Open: func(context.Context, server.Workspace) (server.WorkspaceRuntime, error) {
			t.Error("registration opened a runtime")
			return server.WorkspaceRuntime{}, os.ErrInvalid
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := workspaces.Close(); err != nil {
			t.Error(err)
		}
	})
	f.api.Workspaces = workspaces
	transport := nativeTransport(t, f, "sse")
	project, alias := t.TempDir(), filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(project, alias); err != nil {
		t.Fatal(err)
	}
	canonical, err := filesystem.Canonical(project)
	if err != nil {
		t.Fatal(err)
	}
	identity := ""
	for _, path := range []string{alias, project} {
		reply := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_AddWorkspace{AddWorkspace: &v1.WorkspaceRequest{Path: path}}})
		page := reply.GetWorkspaces()
		if page.GetSelectedId() == "" || identity != "" && identity != page.GetSelectedId() {
			t.Fatal("registration lost its canonical selection", reply)
		}
		identity = page.GetSelectedId()
		found := false
		for _, item := range page.GetItems() {
			if item.GetId() == identity {
				found = item.GetPath() == canonical
			}
		}
		if !found {
			t.Fatal("selection does not identify the registered path", reply)
		}
	}
	if saves != 2 {
		t.Fatal("alias registration wrote the workspace list twice", saves)
	}
}
