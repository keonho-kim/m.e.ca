// Purpose: Prepare a request's explicit or inherited model while keeping the running client's selection stable.
// Pattern: Request admission with conversation-owned client reuse.
// Usage: Start resolves the current default before durable request admission.
// Related: internal/commands/server/tasks.go, internal/commands/server/settings.go, internal/commands/server/default_model.go.
package server

import (
	"github.com/keonho-kim/meca/internal/config"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func (c *Conversation) modelSelection(defaultModel *config.SavedModel) (connection.Selection, error) {
	if !c.InheritsModel {
		return c.Runtime.Selection, nil
	}
	selected, err := configuredSelection(defaultModel)
	if err == nil && sameModelSettings(c.Runtime.Selection, selected) {
		return c.Runtime.Selection, nil
	}
	return selected, err
}

func sameModelSettings(a, b connection.Selection) bool {
	return a.Model == b.Model && a.Connection.ID == b.Connection.ID &&
		a.Connection.CredentialRef == b.Connection.CredentialRef && a.Effort() == b.Effort() &&
		(a.Thinking == b.Thinking || a.Thinking != nil && b.Thinking != nil && *a.Thinking == *b.Thinking)
}

// observedModel requires s.mu. Active requests expose the selection they admitted.
func (s *Service) observedModel(c *Conversation) connection.Selection {
	if !c.InheritsModel || s.sessionRequest(c.Runtime.Session.ID) != nil {
		return c.Runtime.Selection
	}
	selected, _ := c.modelSelection(s.currentConfig().Model)
	return selected
}

func (s *Service) prepareRequestModel(change sessionChange) error {
	c := change.conversation
	if c.Model != nil && !c.InheritsModel {
		return nil
	}
	selected, err := c.modelSelection(change.config.Model)
	if err != nil {
		return err
	}
	if selected.Model == "" {
		return fault.New(diag.FailedPrecondition, diag.ServerSelectAModelBeforeStartingARequest)
	}
	if !c.InheritsModel {
		return s.selectModel(change, ModelSettings{Provider: string(selected.Connection.Kind), Endpoint: selected.Connection.Endpoint, Name: selected.Model, Thinking: selected.Thinking, Effort: selected.ReasoningEffort})
	}
	if c.Model != nil && sameModelSettings(c.Runtime.Selection, selected) {
		return nil
	}
	prepared, err := s.options.PrepareModel(change.ctx, selected, "", modelDefaults(change.config))
	if err != nil {
		return err
	}
	if err := change.ctx.Err(); err != nil {
		if prepared.Close != nil {
			prepared.Close()
		}
		return err
	}
	s.mu.Lock()
	previous := s.installModel(c, prepared, prepared.Selection)
	s.mu.Unlock()
	previous.Close()
	return nil
}
