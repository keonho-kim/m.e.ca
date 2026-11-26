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

func TestDevelopCommandDoesNotRequireProject(t *testing.T) {
	for _, args := range [][]string{
		{"develop", "delayed task queue"},
		{"develop", "delayed", "task", "queue"},
	} {
		mode, err := Parse(args, "/missing/project")
		if err != nil || mode.Action != Develop || mode.DevelopQuery != "delayed task queue" || mode.Project != "" {
			t.Fatalf("develop command was not selected: %+v %v", mode, err)
		}
	}
	mode, err := Parse([]string{"develop", "--help"}, "/missing/project")
	if err != nil || mode.Action != Help {
		t.Fatalf("develop help: %+v %v", mode, err)
	}
	for _, args := range [][]string{{"develop"}, {"develop", " \t "}} {
		if _, err := Parse(args, "/missing/project"); err == nil {
			t.Fatalf("accepted an empty develop query: %v", args)
		}
	}
}

func TestInstallNoShell(t *testing.T) {
	mode, err := Parse([]string{"install", "--no-shell"}, "/missing/project")
	if err != nil || mode.Action != Install || !mode.NoShell || mode.BrowserExtension {
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

func TestBrowserExtensionInstall(t *testing.T) {
	for _, tc := range []struct {
		args    []string
		noShell bool
	}{
		{args: []string{"install", "--ext", "browser"}},
		{args: []string{"install", "--ext=browser"}},
		{args: []string{"install", "--ext", "browser", "--no-shell"}, noShell: true},
		{args: []string{"install", "--no-shell", "--ext=browser"}, noShell: true},
	} {
		mode, err := Parse(tc.args, "/missing/project")
		if err != nil || mode.Action != Install || !mode.BrowserExtension || mode.NoShell != tc.noShell || mode.Project != "" {
			t.Fatalf("%v: %+v %v", tc.args, mode, err)
		}
	}
	for _, args := range [][]string{{"install", "--ext"}, {"install", "--ext", "other"}, {"install", "--ext="}, {"install", "--ext", "browser", "extra"}, {"install", "--ext", "browser", "--ext", "browser"}} {
		if _, err := Parse(args, "/missing/project"); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
