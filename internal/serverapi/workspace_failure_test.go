// Purpose: Check removing a selected path after its file has been deleted.
// Pattern: Native API workflow regression tests.
// Usage: Run go test for this package; failures retain task and storage ownership.
// Related: internal/commands/server/workspace.go.
package serverapi

import (
	"fmt"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"os"
	"path/filepath"
	"testing"
)

func TestDeselectRemovedFile(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	transport := nativeTransport(t, f, "sse")
	state := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_SelectSession{SelectSession: &v1.SelectSession{Create: true}}}).GetState()
	path := filepath.Join(f.project, "removed.txt")
	if err := os.WriteFile(path, []byte("exists"), 0600); err != nil {
		t.Fatal(err)
	}
	selection := &v1.FileSelection{SessionId: state.GetSession().GetId(), Path: "removed.txt", Selected: true}
	if reply := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_SelectFile{SelectFile: selection}}); reply.GetError() != nil {
		t.Fatal(reply)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	selection.Selected = false
	reply := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_SelectFile{SelectFile: selection}})
	if reply.GetError() != nil {
		t.Fatalf("cannot remove stale selection: %v", reply.GetError())
	}
}

func TestMissingHistoryImageReturnsNotFound(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	transport := nativeTransport(t, f, "sse")
	state := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_SelectSession{SelectSession: &v1.SelectSession{Create: true}}}).GetState()
	response := nativeRequest(t, transport, "GET", fmt.Sprintf("/v1/sessions/%s/images/999/0", state.GetSession().GetId()), "")
	if response.Code != 404 {
		t.Fatalf("missing history image returned status=%d body=%s", response.Code, response.Body.String())
	}
}
