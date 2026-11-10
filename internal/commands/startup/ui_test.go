// Purpose: Verify the embedded UI is an explicit server-only startup mode.
// Pattern: Startup boundary contract test.
// Usage: go test ./internal/commands/startup -run TestUI.
// Related: internal/commands/startup/server.go.
package startup

import "testing"

func TestUIRequiresServer(t *testing.T) {
	if _, err := Parse([]string{"--ui"}, t.TempDir()); err == nil {
		t.Fatal("UI must require --server")
	}
	mode, err := Parse([]string{"--server", "--ui"}, t.TempDir())
	if err != nil || !mode.ServerOptions.UI {
		t.Fatalf("server UI selection: %+v %v", mode, err)
	}
	if _, err := Parse([]string{"--server", "--webui"}, t.TempDir()); err == nil {
		t.Fatal("obsolete UI spelling must not be accepted")
	}
}
