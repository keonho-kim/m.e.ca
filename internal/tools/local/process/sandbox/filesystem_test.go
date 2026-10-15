//go:build linux || darwin

// Purpose: Verify native sandbox execution allows project work and denies outside reads and writes.
// Pattern: Native process isolation integration test.
// Usage: go test ./internal/tools/local/process/sandbox on supported Linux or macOS hosts.
// Related: internal/tools/local/process/sandbox/darwin.go.
package sandbox

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestProjectFilesystemIsolation(t *testing.T) {
	requireSandbox(t)
	project, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	private := filepath.Join(project, ".v")
	if err := os.Mkdir(private, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(private, "secret"), []byte("outside-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(outside, "secret")
	if err := os.WriteFile(secret, []byte("outside-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(project, "linked")); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		command string
		denied  bool
	}{
		{"printf inside > result; test -f result", false},
		{"/bin/cat " + strconv.Quote(secret), true},
		{"printf escaped > " + strconv.Quote(filepath.Join(outside, "escape")), true},
		{"/bin/cat linked/secret", true},
		{"/bin/cat .v/secret", true},
		{"printf change > .v/secret", true},
		{"test -f " + strconv.Quote(secret), true},
	} {
		t.Run(tc.command, func(t *testing.T) {
			launch, err := Prepare(context.Background(), project, outside, private, tc.command)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := launch.Close(); err != nil {
					t.Error(err)
				}
			}()
			cmd := exec.Command(launch.Program, launch.Arguments...)
			cmd.Dir = launch.Directory
			cmd.Env = launch.Environment
			cmd.ExtraFiles, cmd.SysProcAttr = launch.Files, launch.Attributes
			if launch.Name != "" {
				cmd.Args[0] = launch.Name
			}
			var output strings.Builder
			cmd.Stdout, cmd.Stderr = &output, &output
			err = cmd.Start()
			if closeErr := launch.CloseInputs(); closeErr != nil {
				t.Fatal(closeErr)
			}
			if err != nil {
				t.Fatal(err)
			}
			err = cmd.Wait()
			if setupErr := launch.SetupError(); setupErr != nil {
				t.Fatal(setupErr)
			}
			if (err != nil) != tc.denied || strings.Contains(output.String(), "outside-secret") {
				t.Fatalf("output=%q err=%v", output.String(), err)
			}
		})
	}
	if _, err := os.Stat(filepath.Join(outside, "escape")); !os.IsNotExist(err) {
		t.Fatalf("outside write succeeded: %v", err)
	}
}
