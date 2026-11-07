// Purpose: Preserve recoverable state before key deletion and serialize resumable uninstall cleanup.
// Pattern: Filesystem and channel-controlled maintenance contract tests.
// Usage: go test ./internal/commands/install -run TestCredentialCleanup.
// Related: internal/commands/install/uninstall.go, internal/commands/install/credential_cleanup.go.
package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func credentialInstallation(t *testing.T) Options {
	t.Helper()
	o := options(t, "/bin/zsh")
	if _, err := Install(t.Context(), o); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(o.Home, ".meca/credentials.used"), []byte("used"), 0600); err != nil {
		t.Fatal(err)
	}
	return o
}

func TestCredentialCleanupFinishesAfterKeyDeletionEvenWhenCanceled(t *testing.T) {
	o := credentialInstallation(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	err := Uninstall(ctx, o.Home, func(context.Context) error {
		if _, err := os.Stat(filepath.Join(o.Home, ".meca")); !errors.Is(err, os.ErrNotExist) {
			t.Error("keys deleted before private state removal", err)
		}
		cancel()
		return nil
	})
	if err != nil {
		t.Fatal("irreversible cleanup stopped after deleting keys", err)
	}
	for _, path := range []string{binaryName, ".meca", cleanupName} {
		if _, err := os.Lstat(filepath.Join(o.Home, path)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("cleanup left %s: %v", path, err)
		}
	}
}

func TestCredentialCleanupCanResumeAndBlocksRestartOrInstall(t *testing.T) {
	o := credentialInstallation(t)
	if err := Uninstall(t.Context(), o.Home, func(context.Context) error { return errors.New("OS store locked") }); err == nil {
		t.Fatal("ignored cleanup failure")
	}
	if err := CheckReady(o.Home); err == nil {
		t.Fatal("restart was allowed during pending cleanup")
	}
	if _, err := Install(t.Context(), o); err == nil {
		t.Fatal("install replaced pending cleanup")
	}
	if _, err := os.Stat(filepath.Join(o.Home, ".meca")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("pending cleanup recreated state", err)
	}
	if err := Uninstall(t.Context(), o.Home, func(context.Context) error { return nil }); err != nil {
		t.Fatal("cleanup could not resume", err)
	}
	if err := CheckReady(o.Home); err != nil {
		t.Fatal(err)
	}
}

func TestCredentialCleanupRejectsConcurrentMaintenance(t *testing.T) {
	o := credentialInstallation(t)
	started, release := make(chan struct{}), make(chan struct{})
	result := make(chan error, 1)
	go func() {
		result <- Uninstall(t.Context(), o.Home, func(context.Context) error { close(started); <-release; return nil })
	}()
	defer func() {
		close(release)
		if err := <-result; err != nil {
			t.Error(err)
		}
	}()
	<-started
	if err := Uninstall(t.Context(), o.Home, func(context.Context) error { t.Error("concurrent OS key deletion"); return nil }); err == nil {
		t.Error("concurrent removal was admitted")
	}
	if _, err := Install(t.Context(), o); err == nil {
		t.Error("concurrent install was admitted")
	}
}
