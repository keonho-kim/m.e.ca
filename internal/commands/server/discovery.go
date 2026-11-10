// Purpose: Discover model metadata without adopting settings or persisting supplied credentials.
// Pattern: Session read use case at the provider I/O boundary.
// Usage: Native model dialogs query candidates before setModel commits a selection.
// Related: internal/provider/setup/setup.go, internal/commands/server/settings.go.
package server

import (
	"context"
	"strings"

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
	c, ctx, done, err := s.beginSessionRead(ctx, value.Session)
	if err != nil {
		return nil, err
	}
	defer done()
	if len(value.Credential) > 2048 || strings.ContainsAny(value.Credential, "\r\n\x00") {
		return nil, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerInvalidCredential)
	}
	selected, err := (config.SavedModel{Provider: connection.ProviderKind(value.Provider), Endpoint: value.Endpoint, Name: value.Name}).Selection()
	// An empty model requests the provider catalog, so parse only its connection.
	if value.Name == "" {
		selected.Connection, err = connection.New(connection.ProviderKind(value.Provider), value.Endpoint)
		selected.Model = ""
	}
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	previous := c.Runtime.Selection
	defaults := s.options.Config
	s.mu.Unlock()
	if selected.Connection.ID == previous.Connection.ID {
		selected.Connection.CredentialRef = previous.Connection.CredentialRef
	}
	values, prepared, err := s.options.Discover(ctx, selected, value.Credential, setup.Defaults{ContextLength: defaults.ContextLength})
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
