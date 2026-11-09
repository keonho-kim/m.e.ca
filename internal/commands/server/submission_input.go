// Purpose: Normalize client history and preserve its transient or durable source ownership.
// Pattern: External input trust and persistence boundary.
// Usage: Compatible submission lends Chat history or stores Responses input once before execution.
// Related: internal/commands/server/submission.go, internal/commands/teamrequest/peer.go.
package server

import (
	"context"
	"encoding/json"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

func seedInput(ctx context.Context, c *Conversation, messages []InputMessage, durable bool) ([]model.Message, error) {
	if len(messages) == 0 {
		return nil, nil
	}
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
	}
	if !durable {
		return contextMessages, ctx.Err()
	}
	// Responses continuations must recover their initial external context after
	// reopening. Store it once in the owning conversation. Team peers receive
	// the public source by reference rather than copying it into every database.
	if c.TeamEnabled {
		body, err := json.Marshal(contextMessages)
		if err != nil {
			return nil, err
		}
		stored, err := c.Team.Repository.AppendTeamMessage(ctx, channel.Message{From: channel.ClientContextAuthor, Observation: true, Text: string(body)})
		if err != nil {
			return nil, err
		}
		return []model.Message{stored.Context()}, nil
	}
	entries := make([]record.Entry, len(contextMessages))
	for i := range contextMessages {
		entries[i].Message = &contextMessages[i]
	}
	return nil, c.Runtime.Store.Commit(ctx, c.Runtime.Session.ID, entries)
}
