//go:build linux

// Purpose: Exercise the actual self-executing Linux sandbox without any external helper.
// Pattern: Native kernel integration harness.
// Usage: V_SANDBOX_REQUIRED=1 makes unavailable kernel isolation a test failure instead of a skip.
// Related: internal/tools/local/process/sandbox/linux/child_linux.go, internal/tools/local/process/sandbox/filesystem_test.go.
package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/keonho-kim/v-code/internal/tools/local/process/sandbox/linux"
)

func TestMain(m *testing.M) {
	if handled, err := linux.Child(); handled {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(125)
	}
	os.Exit(m.Run())
}

func requireSandbox(t *testing.T) {
	t.Helper()
	launch, err := Prepare(context.Background(), t.TempDir(), t.TempDir(), t.TempDir(), "true")
	if err == nil {
		defer func() {
			if err := launch.Close(); err != nil {
				t.Error(err)
			}
		}()
		cmd := exec.Command(launch.Program)
		cmd.Args[0], cmd.ExtraFiles, cmd.SysProcAttr = launch.Name, launch.Files, launch.Attributes
		cmd.Env = launch.Environment
		err = cmd.Start()
		if closeErr := launch.CloseInputs(); closeErr != nil {
			t.Fatal(closeErr)
		}
		if err == nil {
			waitErr := cmd.Wait()
			err = launch.SetupError()
			if err == nil {
				err = waitErr
			}
		}
	}
	if err != nil {
		if os.Getenv("V_SANDBOX_REQUIRED") == "1" {
			t.Fatal(err)
		}
		t.Skipf("kernel sandbox unavailable (use V_SANDBOX_REQUIRED=1 to require it): %v", err)
	}
}

func TestKernelUnavailableDoesNotRunCommand(t *testing.T) {
	if os.Getenv("V_SANDBOX_DENIED") != "1" {
		t.Skip("requires a host policy denying namespaces")
	}
	launch, err := Prepare(context.Background(), t.TempDir(), t.TempDir(), t.TempDir(), "echo unsafe")
	if err != nil {
		return
	}
	defer func() {
		if err := launch.Close(); err != nil {
			t.Error(err)
		}
	}()
	cmd := exec.Command(launch.Program)
	cmd.Args[0], cmd.ExtraFiles, cmd.SysProcAttr = launch.Name, launch.Files, launch.Attributes
	output, err := cmd.CombinedOutput()
	if closeErr := launch.CloseInputs(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if err == nil {
		t.Fatalf("host policy bypassed: %s", output)
	}
	if string(output) == "unsafe\n" {
		t.Fatal("command ran without isolation")
	}
}
