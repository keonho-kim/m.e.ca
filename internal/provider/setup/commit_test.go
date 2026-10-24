// Purpose: Keep session credentials usable when saving the independent new-session default fails.
// Pattern: Persistence failure contract test.
// Usage: go test ./internal/provider/setup -run TestCommit.
// Related: internal/provider/setup/commit.go.
package setup

import (
	"context"
	"errors"
	"testing"

	"github.com/keonho-kim/meca/internal/provider/connection"
)

type commitStore struct {
	selected connection.Selection
	err      error
}

func (s *commitStore) SaveSelection(_ context.Context, _ string, selected connection.Selection) error {
	s.selected = selected
	return s.err
}

type commitVault map[string]string

func (v commitVault) Set(ref, value string) error { v[ref] = value; return nil }

func TestCommitPreservesCredentialWhenDefaultFails(t *testing.T) {
	store, vault := &commitStore{}, commitVault{}
	saved := connection.Selection{Model: "alias"}
	result, err := Commit(context.Background(), store, vault, "session", saved, "secret", func(selected connection.Selection) error {
		if selected.Connection.CredentialRef == "" || selected.Model != store.selected.Model {
			t.Fatal("default saved before session")
		}
		return errors.New("config not writable")
	})
	if err != nil || result.DefaultError == nil || result.Selection.Connection.CredentialRef == "" {
		t.Fatalf("%+v %v", result, err)
	}
	if vault[result.Selection.Connection.CredentialRef] != "secret" {
		t.Fatal("committed session lost its credential")
	}
}

func TestCommitFailureDoesNotPublishDefault(t *testing.T) {
	store, vault := &commitStore{err: errors.New("session write failed")}, commitVault{}
	_, err := Commit(context.Background(), store, vault, "session", connection.Selection{Model: "alias"}, "secret", func(connection.Selection) error { t.Fatal("published failed connection"); return nil })
	if err == nil {
		t.Fatal("ignored session failure")
	}
	for _, value := range vault {
		if value != "" {
			t.Fatal("failed credential was not removed")
		}
	}
}
