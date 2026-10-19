//go:build linux || darwin

// Purpose: Verify cancellation joins the host command and stops its ordinary child process.
// Pattern: Native process lifecycle test.
// Usage: go test ./internal/tools/local/process -run TestHostCancellation.
// Related: internal/tools/local/process/run_unix.go.
package process

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestHostCancellationStopsChild(t *testing.T) {
	project := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	type completion struct {
		output Output
		err    error
	}
	done := make(chan completion, 1)
	go func() {
		output, err := Run(ctx, project, `sleep 60 & child=$!; printf '%s' "$child" > child.pid; wait`, os.Environ())
		done <- completion{output, err}
	}()
	joined := false
	defer func() {
		cancel()
		if !joined {
			<-done
		}
	}()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	var pid int
	for pid == 0 {
		select {
		case result := <-done:
			joined = true
			t.Fatalf("command ended before cancellation: %+v", result)
		case <-ctx.Done():
			t.Fatal("child did not start")
		case <-ticker.C:
			if content, err := os.ReadFile(filepath.Join(project, "child.pid")); err == nil && len(content) > 0 {
				pid, err = strconv.Atoi(string(content))
				if err != nil || pid < 1 {
					t.Fatalf("invalid child PID: %q", content)
				}
			}
		}
	}
	cancel()
	result := <-done
	joined = true
	if !errors.Is(result.err, context.Canceled) || result.output.Status != "interrupted" || !result.output.Started {
		t.Fatalf("cancellation facts lost: %+v", result)
	}
	state, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "stat=").Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	// A defunct child has exited and awaits the host's init reaper.
	if status := strings.TrimSpace(string(state)); status != "" && !strings.HasPrefix(status, "Z") {
		t.Fatalf("child remains running: %d %s", pid, status)
	}
}
