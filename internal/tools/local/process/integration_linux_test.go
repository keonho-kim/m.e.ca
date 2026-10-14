//go:build linux

// Purpose: Verify confined process output, kernel failure reporting, network isolation and cancellation cleanup.
// Pattern: Self-executing Linux integration test.
// Usage: V_SANDBOX_REQUIRED=1 requires kernel support; V_SANDBOX_DENIED=1 verifies host denial.
// Related: internal/tools/local/process/run_unix.go, internal/tools/local/process/sandbox/linux/child_linux.go.
package process

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/keonho-kim/v-code/internal/tools/local/process/sandbox/linux"
	"golang.org/x/sys/unix"
)

func TestMain(m *testing.M) {
	if handled, err := linux.Child(); handled {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(125)
	}
	if len(os.Args) == 2 && os.Args[1] == "probe-kernel" {
		os.Exit(probeKernel())
	}
	if len(os.Args) == 3 && os.Args[1] == "probe-replaced-image" {
		os.Exit(probeReplacedImage(os.Args[2]))
	}
	os.Exit(m.Run())
}

func probeKernel() int {
	for _, address := range []struct{ network, address string }{{"tcp", "127.0.0.1:0"}, {"unix", "host.sock"}} {
		listener, err := net.Listen(address.network, address.address)
		if err == nil {
			if err := listener.Close(); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
			return 1
		}
		if !errors.Is(err, syscall.EPERM) {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
	}
	if !errors.Is(unix.Unshare(unix.CLONE_NEWUSER), unix.EPERM) {
		return 3
	}
	if !errors.Is(unix.Chroot("."), unix.EPERM) {
		return 4
	}
	if err := os.WriteFile("/bin/escaped", []byte("escape"), 0600); err == nil {
		return 5
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		return 6
	}
	nonblockingErr := unix.IoctlSetPointerInt(int(reader.Fd()), 0x5421, 1) // FIONBIO
	deviceErr := unix.IoctlSetPointerInt(int(reader.Fd()), unix.TIOCSTI, 0)
	closeErr := errors.Join(reader.Close(), writer.Close())
	if nonblockingErr != nil || closeErr != nil {
		return 7
	}
	if !errors.Is(deviceErr, unix.EPERM) {
		return 8
	}
	return 0
}

func requireKernel(t *testing.T, project, private string) {
	t.Helper()
	output, err := Run(context.Background(), project, project, private, "true", true)
	if err != nil {
		if output.Status != "scope_unavailable" || os.Getenv("V_SANDBOX_REQUIRED") == "1" {
			t.Fatalf("%+v %v", output, err)
		}
		t.Skipf("kernel isolation unavailable: %v", err)
	}
	if output.ExitCode != 0 {
		t.Fatalf("sandbox initialization failed: %+v", output)
	}
}

func TestConfinedOutputAndKernelRestrictions(t *testing.T) {
	project, private := t.TempDir(), t.TempDir()
	requireKernel(t, project, private)
	output, err := Run(context.Background(), project, project, private, "printf partial; exit 7", true)
	if err != nil || output.Stdout != "partial" || output.ExitCode != 7 || output.Status != "exited" {
		t.Fatalf("%+v %v", output, err)
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(self)
	if err != nil {
		t.Fatal(err)
	}
	target, err := os.OpenFile(filepath.Join(project, "probe"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		t.Fatal(errors.Join(err, source.Close()))
	}
	_, copyErr := io.Copy(target, source)
	if err := errors.Join(copyErr, source.Close(), target.Close()); err != nil {
		t.Fatal(err)
	}
	output, err = Run(context.Background(), project, project, private, "./probe probe-kernel", true)
	if err != nil || output.ExitCode != 0 {
		t.Fatalf("kernel restrictions: %+v %v", output, err)
	}
	// Run outside the sandbox so this parent can replace its on-disk image before
	// launching the next sandbox. The kernel must still execute the original image.
	command := exec.Command(filepath.Join(project, "probe"), "probe-replaced-image", private)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("replaced executable: %s %v", output, err)
	}
}

func probeReplacedImage(private string) int {
	self, err := os.Executable()
	if err != nil {
		return 1
	}
	project := filepath.Dir(self)
	if err := os.Rename(self, self+".old"); err != nil {
		return 2
	}
	if err := os.WriteFile(self, []byte("#!/bin/sh\necho replacement-executed\n"), 0700); err != nil {
		return 3
	}
	output, err := Run(context.Background(), project, project, private, "printf original", true)
	if err != nil || output.Stdout != "original" || output.ExitCode != 0 {
		fmt.Fprintln(os.Stderr, output, err)
		return 4
	}
	return 0
}

func TestConfinedCancellationReapsDetachedChildren(t *testing.T) {
	project, private := t.TempDir(), t.TempDir()
	requireKernel(t, project, private)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type result struct {
		output Output
		err    error
	}
	done := make(chan result, 1)
	marker := "v-orphan-" + filepath.Base(project) + "-" + strconv.Itoa(os.Getpid())
	// setsid escapes the process group, but cannot escape the private PID namespace.
	go func() {
		output, err := Run(ctx, project, project, private, "setsid sh -c 'echo ready; echo ready > ready; sleep 60' "+marker+" & wait", true)
		done <- result{output, err}
	}()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case got := <-done:
			t.Fatalf("premature finish: %+v", got)
		case <-deadline.C:
			cancel()
			<-done
			t.Fatal("shell did not start")
		case <-tick.C:
			if _, err := os.Stat(filepath.Join(project, "ready")); err == nil {
				before, probeErr := matchingProcesses(marker)
				cancel()
				got := <-done
				if probeErr != nil || before == 0 {
					t.Fatalf("detached child not observed: %d %v", before, probeErr)
				}
				if !errors.Is(got.err, context.Canceled) || got.output.Status != "interrupted" || !strings.Contains(got.output.Stdout, "ready") {
					t.Fatalf("%+v", got)
				}
				if after, err := matchingProcesses(marker); err != nil || after != 0 {
					t.Fatalf("children remain after cancellation: %d %v", after, err)
				}
				return
			}
		}
	}
}

func matchingProcesses(marker string) (int, error) {
	paths, err := filepath.Glob("/proc/[0-9]*/cmdline")
	if err != nil {
		return 0, err
	}
	count := 0
	for _, path := range paths {
		command, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return 0, err
		}
		if strings.Contains(string(command), marker) {
			count++
		}
	}
	return count, nil
}

func TestNamespaceDenialReportsScopeFailure(t *testing.T) {
	if os.Getenv("V_SANDBOX_DENIED") != "1" {
		t.Skip("requires host namespace denial")
	}
	project, private := t.TempDir(), t.TempDir()
	output, err := Run(context.Background(), project, project, private, "printf unsafe > "+strconv.Quote(filepath.Join(project, "unsafe")), true)
	if err == nil || output.Status != "scope_unavailable" {
		t.Fatalf("%+v %v", output, err)
	}
	if _, err := os.Stat(filepath.Join(project, "unsafe")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("command was executed: %v", err)
	}
}
