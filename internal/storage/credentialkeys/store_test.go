// Purpose: Verify durable private keys, concurrent publication and rejected unsafe files.
// Pattern: Real-filesystem key-store regression tests.
// Usage: go test -race ./internal/storage/credentialkeys.
// Related: internal/storage/credentialkeys/store.go.
package credentialkeys

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func fixture(t *testing.T) (Store, string, string) {
	t.Helper()
	store := Store{Directory: filepath.Join(t.TempDir(), DirectoryName)}
	ref := "encryption_" + rand.Text()
	value := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))
	return store, ref, value
}

func TestImmutablePrivateKeySurvivesConcurrentPublication(t *testing.T) {
	store, ref, value := fixture(t)
	results := make(chan error, 8)
	for range cap(results) {
		go func() { results <- store.Set(t.Context(), ref, value) }()
	}
	for range cap(results) {
		if err := <-results; err != nil {
			t.Error(err)
		}
	}
	if got, err := store.Get(t.Context(), ref); err != nil || got != value {
		t.Fatal("saved key was not readable", err)
	}
	for _, path := range []string{store.Directory, filepath.Join(store.Directory, Filename)} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm()&0077 != 0 {
			t.Fatal("encryption material is not private", err)
		}
	}
	other := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("v", 32)))
	if err := store.Set(t.Context(), ref, other); err == nil {
		t.Fatal("different material replaced an existing reference")
	}
	if got, err := store.Get(t.Context(), ref); err != nil || got != value {
		t.Fatal("rejected replacement changed an existing key", err)
	}
}

func TestRejectsUnsafeOrDamagedKeyFiles(t *testing.T) {
	for _, problem := range []string{"public", "symlink", "fifo", "corrupt", "oversized", "directory symlink", "public directory"} {
		t.Run(problem, func(t *testing.T) {
			store, ref, value := fixture(t)
			if err := store.Set(t.Context(), ref, value); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(store.Directory, Filename)
			var err error
			switch problem {
			case "public":
				err = os.Chmod(path, 0644)
			case "symlink", "fifo":
				if err = os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if problem == "fifo" {
					err = syscall.Mkfifo(path, 0600)
				} else {
					target := filepath.Join(t.TempDir(), "outside-key")
					if err = os.WriteFile(target, []byte(value), 0600); err != nil {
						t.Fatal(err)
					}
					err = os.Symlink(target, path)
				}
			case "corrupt":
				err = os.WriteFile(path, []byte(strings.Repeat("!", 44)), 0600)
			case "oversized":
				err = os.WriteFile(path, []byte(strings.Repeat(value, 100)), 0600)
			case "directory symlink":
				alias := filepath.Join(t.TempDir(), "alias")
				err = os.Symlink(store.Directory, alias)
				store.Directory = alias
			case "public directory":
				err = os.Chmod(store.Directory, 0755)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.Get(t.Context(), ref); err == nil {
				t.Fatal("unsafe encryption material was accepted")
			}
		})
	}
}

func TestMissingAndCanceledKeysAreNotCreated(t *testing.T) {
	store, ref, value := fixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := store.Set(ctx, ref, value); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := store.Get(t.Context(), ref); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	if _, err := os.Stat(store.Directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("key read or canceled write created a directory")
	}
	for _, ref := range []string{"../outside", "/absolute", "encryption_" + strings.Repeat("/", 26)} {
		if err := store.Set(t.Context(), ref, value); err == nil {
			t.Fatal("unsafe key reference was accepted")
		}
	}
}

func TestKeyDatabaseCrashRecovery(t *testing.T) {
	if directory := os.Getenv("MECA_TEST_KEY_RECOVERY_STATE"); directory != "" {
		store := Store{Directory: directory}
		db, err := store.open(t.Context(), true)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec("PRAGMA cache_size=1; PRAGMA cache_spill=ON; BEGIN IMMEDIATE; UPDATE keys SET material=zeroblob(32)"); err != nil {
			t.Fatal(err)
		}
		for range 4096 {
			if _, err := db.Exec("INSERT INTO keys(ref, material) VALUES (?, zeroblob(32))", "encryption_"+rand.Text()); err != nil {
				t.Fatal(err)
			}
		}
		// Simulate process loss before commit or connection cleanup.
		os.Exit(0)
	}
	store, ref, value := fixture(t)
	if err := store.Set(t.Context(), ref, value); err != nil {
		t.Fatal(err)
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, "-test.run=^TestKeyDatabaseCrashRecovery$")
	command.Env = append(os.Environ(), "MECA_TEST_KEY_RECOVERY_STATE="+store.Directory)
	command.WaitDelay = time.Second
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("interrupted key writer: %v\n%s", err, output)
	}
	if got, err := store.Get(t.Context(), ref); err != nil || got != value {
		t.Fatal("saved key could not be read after an interrupted write", err)
	}
}
