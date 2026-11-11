// Purpose: Discover model metadata without adopting settings or persisting supplied credentials.
// Pattern: Scoped configuration read at the provider I/O boundary.
// Usage: Native model dialogs query candidates before setModel commits a selection.
// Related: internal/provider/setup/setup.go, internal/commands/server/settings.go.
package server

import (
	"context"

	"github.com/keonho-kim/meca/internal/commands/settings"
	"github.com/keonho-kim/meca/internal/config"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/setup"
)

func (s *Service) Discover(ctx context.Context, value ModelSettings) ([]connection.ModelInfo, error) {
	if s.options.Discover == nil {
		return nil, faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerModelDiscoveryUnavailable)
	}
	credential, err := parseModelCredential(value.Credential)
	if err != nil {
		return nil, err
	}
	lookup, err := s.beginModelLookup(ctx, value.Session)
	if err != nil {
		return nil, err
	}
	defer lookup.finish()
	var selected connection.Selection
	// An empty model requests the provider catalog, so parse only its connection.
	if value.Name == "" {
		selected.Connection, err = connection.New(connection.ProviderKind(value.Provider), value.Endpoint)
	} else {
		selected, err = value.selection()
	}
	if err != nil {
		return nil, err
	}
	if selected.Connection.ID == lookup.previous.Connection.ID {
		selected.Connection.CredentialRef = lookup.previous.Connection.CredentialRef
	}
	values, prepared, err := s.options.Discover(lookup.ctx, selected, credential, setup.Defaults{ContextLength: lookup.contextLength})
	if err != nil {
		return nil, err
	}
	if value.Name != "" && prepared != nil && prepared.Details != nil {
		return []connection.ModelInfo{*prepared.Details}, nil
	}
	return values, nil
}

func (s *Service) SetLanguage(ctx context.Context, value string) (Snapshot, error) {
	change, err := s.beginChange(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	defer change.finish()
	next, err := settings.Language(change.ctx, change.config, config.Language(value), s.options.UpdateConfig)
	if err != nil {
		return Snapshot{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.options.Config = next
	s.publish(Event{Record: eventKind("settings_changed")})
	return s.snapshot(), nil
}
