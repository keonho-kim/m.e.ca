// Purpose: Verify mutation targets remain owned through queued commands and process cleanup.
// Pattern: Real process-registry and workspace integration tests.
// Usage: go test -race ./internal/tools/team -run TestCommandLease.
// Related: internal/tools/team/command_admission.go, internal/tools/local/process/command_start.go.
package team

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/tools/local/process"
	"github.com/keonho-kim/meca/internal/tools/team/workspace"
)

func TestCommandLeaseAdmissionIsObservableAndShutdownReleasesTargets(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	w := workspace.New()
	paths := []string{filepath.Join(t.TempDir(), "file")}
	first, err := w.ReserveCommand(ctx, paths, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Release()
	var r process.Registry
	r.Begin(ctx, time.Now().Add(time.Minute))
	defer r.End()
	entered := make(chan struct{})
	id, err := r.StartAdmitted(ctx, t.TempDir(), "printf done", os.Environ(), process.CallIdentity{CallID: "call"}, func(ctx context.Context) (func(), error) {
		held, err := w.ReserveCommand(ctx, paths, nil, func() error { close(entered); return nil })
		if err != nil {
			return nil, err
		}
		return held.Release, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	<-entered
	state, err := r.Wait(ctx, id, 0)
	if err != nil || state.Status != "waiting" {
		t.Fatalf("queued command not visible: %+v %v", state, err)
	}
	first.Release()
	state, err = r.Wait(ctx, id, time.Minute)
	if err != nil || state.ExitCode != 0 {
		t.Fatalf("queued command failed: %+v %v", state, err)
	}
	held, err := w.ReserveFiles(ctx, paths, nil, func() error { t.Error("completed command retained lease"); return context.Canceled })
	if err != nil {
		t.Fatal(err)
	}
	held.Release()
}
func TestCommandLeaseCancellationDoesNotReleaseAnotherCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	w := workspace.New()
	path := filepath.Join(t.TempDir(), "file")
	first, _ := w.ReserveCommand(ctx, []string{path}, nil, nil)
	defer first.Release()
	var r process.Registry
	r.Begin(ctx, time.Now().Add(time.Minute))
	entered := make(chan struct{})
	_, err := r.StartAdmitted(ctx, t.TempDir(), "printf never", os.Environ(), process.CallIdentity{CallID: "call"}, func(ctx context.Context) (func(), error) {
		held, err := w.ReserveCommand(ctx, []string{path}, nil, func() error { close(entered); return nil })
		if err != nil {
			return nil, err
		}
		return held.Release, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	<-entered
	r.End()
	probe, stop := context.WithCancel(ctx)
	held, err := w.ReserveFiles(probe, []string{path}, nil, func() error { stop(); return nil })
	if err == nil || held != nil {
		t.Fatal("cancelling a queued command released another command's target")
	}
}
