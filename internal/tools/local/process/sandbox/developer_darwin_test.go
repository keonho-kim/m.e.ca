//go:build darwin

// Purpose: Verify selected Apple tools work inside project containment and retain isolation.
// Pattern: Native platform integration tests.
// Usage: go test ./internal/tools/local/process/sandbox -run TestApple.
// Related: internal/tools/local/process/sandbox/developer_darwin.go.
package sandbox

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAppleToolsInSandbox(t *testing.T) {
	if _, err := exec.Command("/usr/bin/xcode-select", "-p").Output(); err != nil {
		t.Skip("Apple developer tools are not installed")
	}
	project, home := t.TempDir(), t.TempDir()
	private := filepath.Join(home, ".v")
	if err := os.Mkdir(private, 0700); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"python3 -c 'import sys,sysconfig,tempfile; print(sys.version); print(sysconfig.get_paths())'", "git --version", "xcrun --find python3", "printf 'int main(void) {return 0;}' | clang -x c -fsyntax-only -"} {
		launch, err := Prepare(context.Background(), project, home, private, command)
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(launch.Program, launch.Arguments...)
		cmd.Dir = launch.Directory
		cmd.Env = launch.Environment
		output, runErr := cmd.CombinedOutput()
		if err := launch.Close(); err != nil {
			t.Fatal(err)
		}
		if runErr != nil {
			t.Errorf("%s: %s %v", command, output, runErr)
		}
	}
}

func TestAppleDeveloperRootCannotExposeHomeOrPrivate(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	private := filepath.Join(home, ".v")
	if err := os.Mkdir(private, 0700); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/", home, private, project, filepath.Join(home, "missing")} {
		t.Setenv("DEVELOPER_DIR", path)
		if _, err := developerRuntime(context.Background(), project, home, private); err == nil {
			t.Fatalf("accepted broad or invalid root %s", path)
		}
	}
}

func TestAppleDiscoveryCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := developerRuntime(ctx, t.TempDir(), t.TempDir(), t.TempDir()); err != context.Canceled {
		t.Fatalf("cancellation lost: %v", err)
	}
}
