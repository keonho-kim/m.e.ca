// Purpose: Verify executable argument parsing and project selection.
// Pattern: Behavioral tests.
// Usage: go test ./internal/commands/startup.
// Related: internal/commands/startup/parse.go.
package startup

import (
	"path/filepath"
	"testing"
)

func TestStartModes(t *testing.T) {
	root := t.TempDir()
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		args   []string
		resume bool
		fail   bool
	}{{nil, false, false}, {[]string{root}, false, false}, {[]string{"resume"}, true, false}, {[]string{"missing"}, false, true}, {[]string{"--bad"}, false, true}, {[]string{root, root}, false, true}} {
		mode, err := Parse(tc.args, root)
		if (err != nil) != tc.fail {
			t.Fatalf("%v: %v", tc.args, err)
		}
		if err == nil && (mode.Resume != tc.resume || !tc.resume && mode.Project != real) {
			t.Fatalf("%v: %+v", tc.args, mode)
		}
	}
}

func TestMaintenanceDoesNotRequireProjectDirectory(t *testing.T) {
	for _, command := range []Action{Install, Uninstall, "upgrade"} {
		mode, err := Parse([]string{string(command)}, "/missing/project")
		if err != nil || mode.Action != command || mode.Resume || mode.Project != "" {
			t.Fatalf("%s: %+v %v", command, mode, err)
		}
	}
}

func TestInstallNoShell(t *testing.T) {
	mode, err := Parse([]string{"install", "--no-shell"}, "/missing/project")
	if err != nil || mode.Action != Install || !mode.NoShell {
		t.Fatalf("mode=%+v err=%v", mode, err)
	}
	for _, args := range [][]string{{"install", "extra"}, {"install", "--unknown"}, {"--no-shell"}} {
		if _, err := Parse(args, t.TempDir()); err == nil {
			t.Fatalf("unsupported install arguments accepted: %v", args)
		}
	}
}

func TestUpgradeRejectsArguments(t *testing.T) {
	for _, args := range [][]string{{"upgrade", "extra"}, {"upgrade", "--print"}, {"upgrade", "--no-shell"}} {
		if _, err := Parse(args, "/missing/project"); err == nil {
			t.Fatalf("unsupported upgrade arguments accepted: %v", args)
		}
	}
}
