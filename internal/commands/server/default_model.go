// Purpose: Read and commit the server's global default model independently of conversation storage.
// Pattern: Configuration transaction with a request-owned provider client.
// Usage: Native model settings explicitly select this scope.
// Related: internal/commands/server/settings.go, internal/commands/modelchange/commit.go, internal/config/update.go.
package server

import (
	"context"

	"github.com/keonho-kim/meca/internal/commands/modelchange"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func configuredSelection(model *config.SavedModel) (connection.Selection, error) {
	if model == nil {
		return connection.Selection{}, nil
	}
	return model.Selection()
}

func (s *Service) DefaultModel(ctx context.Context) (Snapshot, error) {
	lookup, err := s.beginModelLookup(ctx, "")
	if err != nil {
		return Snapshot{}, err
	}
	defer lookup.finish()
	return Snapshot{Selection: lookup.previous}, nil
}

func (s *Service) setDefaultModel(ctx context.Context, value ModelSettings) (Snapshot, error) {
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	defer done()
	value.Credential, err = parseModelCredential(value.Credential)
	if err != nil {
		return Snapshot{}, err
	}
	s.mu.Lock()
	cfg := s.currentConfig()
	s.mu.Unlock()
	current, err := configuredSelection(cfg.Model)
	if err != nil {
		return Snapshot{}, err
	}
	requested, err := value.selection()
	if err != nil {
		return Snapshot{}, err
	}
	if requested.Connection.ID == current.Connection.ID && value.Credential == "" {
		requested.Connection.CredentialRef = current.Connection.CredentialRef
		if sameModelSettings(current, requested) {
			return Snapshot{Selection: current}, nil
		}
	}
	prepared, err := s.prepareModelChange(ctx, cfg, value, current)
	if err != nil {
		return Snapshot{}, err
	}
	if prepared.Close != nil {
		defer prepared.Close()
	}
	selected, err := modelchange.BindCredential(ctx, s.options.ModelCredentials, current, prepared.Selection, value.Credential)
	if err != nil {
		return Snapshot{}, err
	}
	next, err := s.options.UpdateConfig(ctx, func(latest *config.Config) {
		latest.Model = config.RememberModel(selected)
	})
	if err != nil {
		return Snapshot{}, err
	}
	s.mu.Lock()
	s.options.Config = next
	s.publish(Event{Record: eventKind("settings_changed")})
	s.mu.Unlock()
	return Snapshot{Selection: selected}, nil
}
