// Purpose: Prepare, durably commit and adopt idle-only model and permission changes.
// Pattern: Reserved state transition around existing configuration and model I/O use cases.
// Usage: SetModel and Configure keep network, keyring and persistence work outside the state mutex.
// Related: internal/commands/modelchange/commit.go, internal/commands/server/operations.go.
package server

import (
	"context"
	"strings"

	conversationruntime "github.com/keonho-kim/meca/internal/commands/conversation"
	"github.com/keonho-kim/meca/internal/commands/modelchange"

	"github.com/keonho-kim/meca/internal/commands/settings"
	"github.com/keonho-kim/meca/internal/config"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/setup"
)

func (s *Service) Configure(ctx context.Context, permission *string, team *bool) (Snapshot, error) {
	if (permission == nil) == (team == nil) {
		return Snapshot{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerChangeExactlyOneSettingPerCommand)
	}
	change, err := s.beginChange(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	defer change.finish()
	c := change.conversation
	if c == nil {
		return Snapshot{}, faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerOpenASessionFirst)
	}
	next := change.config
	if permission != nil {
		var value policy.Permission
		if err := value.UnmarshalText([]byte(*permission)); err != nil {
			return Snapshot{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.WrappedCause, err)
		}
		next, err = settings.Permission(change.ctx, next, value, s.options.UpdateConfig)
	} else {
		if *team && c.Team == nil {
			return Snapshot{}, faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerTeamServiceIsUnavailable)
		}
		err = c.History.SetTeamMode(change.ctx, *team)
	}
	if err != nil {
		return Snapshot{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.options.Config = next
	if team != nil {
		c.TeamEnabled = *team
	}
	s.publish(Event{Record: eventKind("settings_changed")})
	return s.snapshot(), nil
}

func (s *Service) SetModel(ctx context.Context, value ModelSettings) (Snapshot, error) {
	change, err := s.beginChange(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	defer change.finish()
	if change.conversation == nil {
		return Snapshot{}, faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerOpenASessionFirst)
	}
	err = s.selectModel(change, value)
	return s.State(), err
}

// selectModel requires a change reservation. Commit owns partial-default failure semantics.
func (s *Service) selectModel(change sessionChange, value ModelSettings) error {
	c := change.conversation
	if c.Models == nil || c.Runtime.Vault == nil {
		return faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerSessionModelPersistenceIsUnavailable)
	}
	if len(value.Credential) > 2048 || strings.ContainsAny(value.Credential, "\r\n\x00") {
		return faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerInvalidCredential)
	}
	value.Credential = strings.Trim(value.Credential, " \t")
	if err := c.Runtime.Vault.Remember(value.Credential); err != nil {
		return err
	}
	selected, err := (config.SavedModel{Provider: connection.ProviderKind(value.Provider), Endpoint: value.Endpoint, Name: value.Name, Thinking: value.Thinking, ReasoningEffort: value.Effort}).Selection()
	if err != nil {
		return faults.New(diagcatalog.InvalidArgument, diagcatalog.WrappedCause, err)
	}
	if value.Credential == "" && c.Runtime.Selection.Connection.ID == selected.Connection.ID {
		selected.Connection.CredentialRef = c.Runtime.Selection.Connection.CredentialRef
	}
	prepared, err := s.options.PrepareModel(change.ctx, selected, value.Credential, setup.Defaults{ContextLength: change.config.ContextLength, Thinking: change.config.Thinking})
	if err != nil {
		return err
	}
	adopted := false
	defer func() {
		if !adopted && prepared.Close != nil {
			prepared.Close()
		}
	}()
	if err := change.ctx.Err(); err != nil {
		return err
	}
	var remember func(connection.Selection) error
	next := change.config
	if value.Remember {
		remember = func(selection connection.Selection) error {
			var err error
			next, err = s.options.UpdateConfig(change.ctx, func(latest *config.Config) {
				latest.Model = config.RememberModel(selection)
			})
			return err
		}
	}
	committed, err := modelchange.Commit(change.ctx, c.Models, c.Runtime.Vault, c.Runtime.Session.ID, prepared.Selection, value.Credential, remember)
	if err != nil {
		if committed.Uncertain {
			s.mu.Lock()
			previous := c.Model
			c.Model, c.Runtime.Selection = nil, connection.Selection{}
			s.publish(Event{Session: c.Runtime.Session.ID, Record: eventKind("model_unavailable")})
			s.mu.Unlock()
			previous.Close()
		}
		return err
	}
	s.mu.Lock()
	previous := c.Model
	c.Runtime.Selection, c.Model = committed.Selection, conversationruntime.NewModel(prepared.Generator, prepared.Close)
	if value.Remember && committed.DefaultError == nil {
		s.options.Config = next
	}
	s.publish(Event{Session: c.Runtime.Session.ID, Record: eventKind("model_changed")})
	s.mu.Unlock()
	adopted = true
	previous.Close()
	return committed.DefaultError
}
