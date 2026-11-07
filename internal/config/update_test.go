// Purpose: Preserve independent settings across concurrent processes and canceled lock acquisition.
// Pattern: Filesystem configuration transaction tests.
// Usage: go test -race ./internal/config -run TestUpdate.
// Related: internal/config/update.go, internal/config/statefs/lease_unix.go.
package config

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/keonho-kim/meca/internal/policy"
)

func TestUpdatePreservesIndependentWriters(t *testing.T) {
	directory := t.TempDir()
	var wg sync.WaitGroup
	for _, apply := range []func(*Config){
		func(c *Config) { c.Language = LanguageEnglish },
		func(c *Config) { c.Permission = policy.PermissionAllMighty },
	} {
		wg.Go(func() {
			if _, err := Update(t.Context(), directory, apply); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	got, err := Load(directory)
	if err != nil || got.Language != LanguageEnglish || got.Permission != policy.PermissionAllMighty {
		t.Fatal("independent configuration update was lost", got, err)
	}
}

func TestUpdateCanceledAndInvalidPreserveFile(t *testing.T) {
	directory := t.TempDir()
	if err := Save(directory, Defaults()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	called := false
	if _, err := Update(ctx, directory, func(c *Config) { called = true }); !errors.Is(err, context.Canceled) || called {
		t.Fatal("canceled update reached mutation", err)
	}
	if _, err := Update(t.Context(), directory, func(c *Config) { c.Language = "invalid" }); err == nil {
		t.Fatal("invalid preference was saved")
	}
	got, err := Load(directory)
	if err != nil || got.Language != LanguageAuto {
		t.Fatal("failed update changed configuration", err)
	}
}
