// Purpose: Isolate compatible histories and reject stale response continuations before execution.
// Pattern: Conversation ownership and input trust boundary.
// Usage: Submit holds one change reservation across session transfer and history seeding.
// Related: internal/commands/server/lifecycle.go, internal/agent/context/conversation.go.
package server

import (
	"context"
	"errors"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
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
			return nil, failure("conflict", "previous_response_id must name the latest completed response")
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
		return nil, failure("failed_precondition", "conversation result history is unavailable")
	}
	if previous != "" {
		position, err := c.Output.OutputPosition(change.ctx, c.Runtime.Session.ID)
		if err != nil {
			return nil, err
		}
		if position != parent.Through {
			return nil, failure("conflict", "conversation changed after previous_response_id")
		}

	}
	if c.TeamEnabled && c.Team == nil {
		return nil, failure("failed_precondition", "team service is unavailable")
	}
	if c.Model == nil {
		selected := c.Runtime.Selection
		if selected.Model == "" {
			return nil, failure("failed_precondition", "select or save a model before using the compatible API")
		}
		change.conversation = c
		if err := s.selectModel(change, ModelSettings{Provider: string(selected.Connection.Kind), Endpoint: selected.Connection.Endpoint, Name: selected.Model, Thinking: selected.Thinking, Effort: selected.ReasoningEffort}); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func seedInput(ctx context.Context, c *Conversation, messages []InputMessage) ([]model.Message, error) {
	if len(messages) == 0 {
		return nil, nil
	}
	entries := make([]record.Entry, len(messages))
	contextMessages := make([]model.Message, len(messages))
	for i, input := range messages {
		message := model.Message{Role: input.Role, Content: c.Runtime.Vault.Redact(input.Text)}
		if input.Role == "system" || input.Role == "developer" {
			// Client instructions are contextual notes. The runtime's operator
			// instructions and actual execution/approval policy retain ownership.
			message.Role = "system"
			message.Content = "Client " + input.Role + " instructions:\n" + message.Content
		}
		contextMessages[i] = message
		entries[i].Message = &contextMessages[i]
	}
	return contextMessages, c.Runtime.Store.Commit(ctx, c.Runtime.Session.ID, entries)
}
