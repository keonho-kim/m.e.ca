// Purpose: Verify terminal-free invocation, exact session selection and export parsing.
// Pattern: CLI boundary tests.
// Usage: go test ./internal/commands/startup.
// Related: internal/commands/startup/parse.go, cmd/v/headless.go.
package startup

import "testing"

func TestHeadlessModes(t *testing.T) {
	root := t.TempDir()
	for _, args := range [][]string{{"--print", "Create a file"}, {root, "--mode", "json", "Inspect"}, {"--session", "session_example", "--export"}, {"--diagnose"}} {
		mode, err := Parse(args, root)
		if err != nil || !mode.Print {
			t.Fatalf("%v: %+v %v", args, mode, err)
		}
	}
	for _, args := range [][]string{{"--mode", "xml"}, {"--export"}, {"--diagnose", "unexpected prompt"}} {
		if _, err := Parse(args, root); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
