// Purpose: Acquire the compatible conversation and reject stale continuation boundaries before execution.
// Pattern: Conversation ownership transition.
// Usage: Submit holds one change reservation across session transfer and history seeding.
// Related: internal/commands/server/lifecycle.go, internal/agent/context/conversation.go.
package server

import (
	"errors"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *Service) submissionSession(change sessionChange, previous, id string) (*Conversation, error) {
	var parent Response
	var err error
	if previous != "" {
		parent, err = s.ResponseState(change.ctx, previous, true)
		if err != nil {
			return nil, err
		}
		if parent.Status != "completed" || parent.Next != "" && parent.Next != id {
			return nil, faults.New(diagcatalog.Conflict, diagcatalog.ServerPreviousResponseIdMustNameTheLatest)
		}
	}
	s.mu.Lock()
	cached := s.job
	s.mu.Unlock()
	var c *Conversation
	if previous != "" {
		if change.conversation != nil && change.conversation.Runtime.Session.ID == parent.Session {
			c = change.conversation
		}
		if c == nil && cached != nil && cached.Runtime.Session.ID == parent.Session {
			c = cached
		}
	}
	if c == nil {
		next, err := s.options.Open(change.ctx, parent.Session, previous == "", change.config)
		if err != nil {
			return nil, err
		}
		donor := change.conversation
		if donor == nil {
			donor = cached
		}
		if previous == "" && donor != nil {
			next.Runtime.Selection = donor.Runtime.Selection
			if next.TeamEnabled != donor.TeamEnabled {
				err = next.History.SetTeamMode(change.ctx, donor.TeamEnabled)
				next.TeamEnabled = donor.TeamEnabled
			}
			if err == nil {
				err = s.transferModel(change.ctx, donor, next)
			}
			if err != nil {
				return nil, errors.Join(err, closeConversation(next))
			}
		}
		if err := s.replaceJob(change, next); err != nil {
			return nil, err
		}
		c = next
	}

	if c.Output == nil {
		return nil, faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerConversationResultHistoryIsUnavailable)
	}
	if previous != "" {
		position, err := c.Output.OutputPosition(change.ctx, c.Runtime.Session.ID)
		if err != nil {
			return nil, err
		}
		if position != parent.Through {
			return nil, faults.New(diagcatalog.Conflict, diagcatalog.ServerConversationChangedAfterPreviousResponseId)
		}

	}
	if c.TeamEnabled && c.Team == nil {
		return nil, faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerTeamServiceIsUnavailable)
	}
	if c.Model == nil {
		selected := c.Runtime.Selection
		if selected.Model == "" {
			return nil, faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerSelectOrSaveAModelBeforeUsing)
		}
		change.conversation = c
		if err := s.selectModel(change, ModelSettings{Provider: string(selected.Connection.Kind), Endpoint: selected.Connection.Endpoint, Name: selected.Model, Thinking: selected.Thinking, Effort: selected.ReasoningEffort}); err != nil {
			return nil, err
		}
	}
	return c, nil
}
