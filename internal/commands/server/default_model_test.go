// Purpose: Verify global model saving and lifecycle without a selected conversation.
// Pattern: Server configuration ownership regression test.
// Usage: go test ./internal/commands/server -run TestDefaultModel.
// Related: internal/commands/server/settings.go, internal/commands/server/default_model.go.
package server

import (
	"context"
	"errors"
	"testing"

	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/setup"
)

type defaultModelVault struct {
	ref, scope, credential string
	writes                 int
}

func (v *defaultModelVault) Set(_ context.Context, ref, scope, credential string) error {
	v.ref, v.scope, v.credential = ref, scope, credential
	v.writes++
	return nil
}

func (v *defaultModelVault) Get(_ context.Context, ref, scope string) (string, error) {
	if ref != v.ref || scope != v.scope {
		return "", errors.New("unexpected credential reference")
	}
	return v.credential, nil
}

func TestDefaultModelSavesWithoutConversation(t *testing.T) {
	s := detailFixture(&detailHistory{}, &credentials.Vault{})
	defer s.Close()
	if err := s.CloseSession(t.Context(), s.State().Session.ID); err != nil {
		t.Fatal(err)
	}
	var saved config.Config
	closed := 0
	vault := &defaultModelVault{}
	s.options.ModelCredentials = vault
	s.options.PrepareModel = func(_ context.Context, value connection.Selection, _ string, _ setup.Defaults) (PreparedModel, error) {
		return PreparedModel{Selection: value, Close: func() { closed++ }}, nil
	}
	s.options.UpdateConfig = func(_ context.Context, apply func(*config.Config)) (config.Config, error) {
		apply(&saved)
		return saved, nil
	}
	input := ModelSettings{Provider: "ollama", Name: "global-model", Credential: "fixture-key", DefaultModel: true}
	reply, err := s.SetModel(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Model == nil || saved.Model.Name != "global-model" || reply.Selection.Model != "global-model" {
		t.Fatalf("global selection was not saved and returned: saved=%v reply=%q", saved.Model, reply.Selection.Model)
	}
	if reply.Session.ID != "" || s.State().Session.ID != "" || closed != 1 {
		t.Fatalf("default save acquired a session or retained a model client: session=%q closes=%d", reply.Session.ID, closed)
	}
	if saved.Model.CredentialRef == "" || saved.Model.CredentialRef != vault.ref || vault.credential != input.Credential {
		t.Fatal("global default did not store a reference to its credential")
	}
	if _, err := s.SetModel(t.Context(), input); err != nil || vault.writes != 1 {
		t.Fatalf("saving the same global credential wrote a duplicate: writes=%d error=%v", vault.writes, err)
	}
}

func TestDefaultModelRefreshesInheritedConversationOnAdmission(t *testing.T) {
	s := detailFixture(&detailHistory{}, &credentials.Vault{})
	defer s.Close()
	c := s.conversation
	c.InheritsModel = true
	s.options.Config.Model = &config.SavedModel{Provider: connection.Ollama, Name: "before"}
	s.options.PrepareModel = func(_ context.Context, value connection.Selection, _ string, _ setup.Defaults) (PreparedModel, error) {
		return PreparedModel{Selection: value}, nil
	}
	s.options.UpdateConfig = func(_ context.Context, apply func(*config.Config)) (config.Config, error) {
		next := s.options.Config
		apply(&next)
		return next, nil
	}
	prepare := func(expected string) {
		t.Helper()
		change, err := s.beginChange(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		defer change.finish()
		if err := s.prepareRequestModel(change); err != nil || c.Runtime.Selection.Model != expected {
			t.Fatalf("request model=%q, expected=%q, error=%v", c.Runtime.Selection.Model, expected, err)
		}
	}
	prepare("before")
	if _, err := s.SetModel(t.Context(), ModelSettings{Provider: "ollama", Name: "after", DefaultModel: true}); err != nil {
		t.Fatal(err)
	}
	if c.Runtime.Selection.Model != "before" || s.State().Selection.Model != "after" {
		t.Fatal("default save mutated the prepared client or exposed stale state")
	}
	prepare("after")
	if saved, err := c.Models.Selection(t.Context(), c.Runtime.Session.ID); err != nil || saved.Model != "" {
		t.Fatalf("inherited model became an explicit session selection: %q, %v", saved.Model, err)
	}
}

func TestDefaultModelPreparationDoesNotReserveConversation(t *testing.T) {
	s := detailFixture(&detailHistory{}, &credentials.Vault{})
	defer s.Close()
	entered, release := make(chan struct{}), make(chan struct{})
	s.options.PrepareModel = func(ctx context.Context, value connection.Selection, _ string, _ setup.Defaults) (PreparedModel, error) {
		close(entered)
		select {
		case <-release:
			return PreparedModel{Selection: value}, nil
		case <-ctx.Done():
			return PreparedModel{}, ctx.Err()
		}
	}
	s.options.UpdateConfig = func(_ context.Context, apply func(*config.Config)) (config.Config, error) {
		var next config.Config
		apply(&next)
		return next, nil
	}
	done := make(chan error, 1)
	go func() {
		_, err := s.SetModel(t.Context(), ModelSettings{Provider: "ollama", Name: "next", DefaultModel: true})
		done <- err
	}()
	<-entered
	_, selectErr := s.Select(t.Context(), s.State().Session.ID, false)
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if selectErr != nil {
		t.Fatalf("global provider probe blocked conversation selection: %v", selectErr)
	}
}

func TestDefaultModelUnchangedSaveSkipsProviderAndPersistence(t *testing.T) {
	s := detailFixture(&detailHistory{}, &credentials.Vault{})
	defer s.Close()
	s.options.Config.Model = &config.SavedModel{Provider: connection.Ollama, Name: "unchanged"}
	s.options.PrepareModel = func(context.Context, connection.Selection, string, setup.Defaults) (PreparedModel, error) {
		t.Fatal("unchanged default probed provider")
		return PreparedModel{}, nil
	}
	s.options.UpdateConfig = func(context.Context, func(*config.Config)) (config.Config, error) {
		t.Fatal("unchanged default wrote configuration")
		return config.Config{}, nil
	}
	result, err := s.SetModel(t.Context(), ModelSettings{Provider: "ollama", Name: "unchanged", DefaultModel: true})
	if err != nil || result.Selection.Model != "unchanged" {
		t.Fatalf("unchanged default: %q %v", result.Selection.Model, err)
	}
}
