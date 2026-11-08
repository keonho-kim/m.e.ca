// Purpose: Keep session credentials usable when saving the independent new-session default fails.
// Pattern: Persistence failure contract test.
// Usage: go test ./internal/provider/setup -run TestCommit.
// Related: internal/commands/modelchange/commit.go.
package modelchange

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

func TestCommitWhitespaceDoesNotPublishMissingCredential(t *testing.T) {
	store, vault := &commitStore{}, commitVault{}
	result, err := Commit(t.Context(), store, vault, "session", connection.Selection{Model: "alias"}, " \t ", nil)
	if err != nil || result.Selection.Connection.CredentialRef != "" || len(vault) != 0 {
		t.Fatal("empty credential published a dangling reference", err)
	}
}

func (s *commitStore) SaveSelection(_ context.Context, _ string, selected connection.Selection) error {
	s.selected = selected
	return s.err
}

type commitVault map[string]string

func (v commitVault) Set(_ context.Context, ref, scope, value string) error {
	v[ref] = value
	return nil
}

func (v commitVault) Get(_ context.Context, ref, _ string) (string, error) {
	return v[ref], nil
}

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
	if vault[store.selected.Connection.CredentialRef] != "secret" {
		t.Fatal("uncertain selection commit lost its credential")
	}
}

func (s *commitStore) Selection(context.Context, string) (connection.Selection, error) {
	if s.err != nil {
		return connection.Selection{}, nil
	}
	return s.selected, nil
}

func TestCanceledCommitDoesNotWriteCredentials(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	store, writer := &commitStore{}, commitVault{}
	if _, err := Commit(ctx, store, writer, "session", connection.Selection{Model: "alias"}, "secret", nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if len(writer) != 0 || store.selected.Model != "" {
		t.Fatal("cancelled commit performed persistence")
	}
}
