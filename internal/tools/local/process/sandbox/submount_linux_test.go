//go:build linux

// Purpose: Reproduce locked runtime submounts while checking wc execution and hidden host content.
// Pattern: Native Linux namespace regression test.
// Usage: Set V_TEST_NESTED_RUNTIME=1 after mounting a fixture at /usr/lib/v-wsl-test.
// Related: internal/tools/local/process/sandbox/linux/runtime_linux.go.
package sandbox

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeNestedMountIsolation(t *testing.T) {
	if os.Getenv("V_TEST_NESTED_RUNTIME") != "1" {
		t.Skip("requires a nested runtime mount fixture")
	}
	project, home := t.TempDir(), t.TempDir()
	private := filepath.Join(home, ".v")
	if err := os.Mkdir(private, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "input"), []byte("one\ntwo\n"), 0600); err != nil {
		t.Fatal(err)
	}
	launch, err := prepareTest(context.Background(), project, home, private, "wc -l input && if test -r /usr/lib/v-wsl-test/secret || test -r /usr/lib/v-wsl-test/nested/secret; then echo exposed; exit 9; fi")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := launch.Close(); err != nil {
			t.Error(err)
		}
	}()
	cmd := exec.Command(launch.Program, launch.Arguments...)
	cmd.Args[0] = launch.Name
	cmd.Dir, cmd.Env, cmd.ExtraFiles, cmd.SysProcAttr = launch.Directory, launch.Environment, launch.Files, launch.Attributes
	var output strings.Builder
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if err := launch.CloseInputs(); err != nil {
		t.Fatal(err)
	}
	err = cmd.Wait()
	if setup := launch.SetupError(); setup != nil {
		t.Fatal(setup)
	}
	if err != nil || !strings.Contains(output.String(), "2") || strings.Contains(output.String(), "exposed") {
		t.Fatalf("%q %v", output.String(), err)
	}
}
