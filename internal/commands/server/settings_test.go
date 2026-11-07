// Purpose: Preserve remembered model defaults across later permission changes and partial save failures.
// Pattern: Configuration ownership contract tests.
// Usage: go test ./internal/commands/server -run TestRememberedModel.
// Related: internal/commands/server/service.go, internal/config/settings.go.
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

func TestRememberedModelSurvivesPermissionChanges(t *testing.T) {
	s := detailFixture(&detailHistory{}, &credentials.Vault{})
	defer s.Close()
	conn, err := connection.New(connection.Ollama, "")
	if err != nil {
		t.Fatal(err)
	}
	selected := connection.Selection{Connection: conn, Model: "first"}
	s.options.PrepareModel = func(context.Context, connection.Selection, string, setup.Defaults) (PreparedModel, error) {
		return PreparedModel{Selection: selected}, nil
	}
	var saved config.Config
	fail := false
	s.options.UpdateConfig = func(_ context.Context, apply func(*config.Config)) (config.Config, error) {
		next := s.options.Config
		apply(&next)
		if fail {
			return config.Config{}, errors.New("disk unavailable")
		}
		saved = next
		return next, nil
	}
	if _, err := s.SetModel(t.Context(), ModelSettings{Provider: "ollama", Name: "fixture", Remember: true}); err != nil {
		t.Fatal(err)
	}
	permission := "all_mighty"
	if _, err := s.Configure(t.Context(), &permission, nil); err != nil {
		t.Fatal(err)
	}
	if saved.Model == nil || saved.Model.Name != "first" {
		t.Fatal("permission change discarded remembered model", saved.Model)
	}
	fail = true
	selected.Model = "second"
	if _, err := s.SetModel(t.Context(), ModelSettings{Provider: "ollama", Name: "fixture", Remember: true}); err == nil {
		t.Fatal("default failure was hidden")
	}
	if s.State().Selection.Model != "second" {
		t.Fatal("committed session model was discarded")
	}
	fail = false
	permission = "default"
	if _, err := s.Configure(t.Context(), &permission, nil); err != nil {
		t.Fatal(err)
	}
	if saved.Model.Name != "first" {
		t.Fatal("failed default save silently changed global selection", saved.Model)
	}
}
