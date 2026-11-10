// Purpose: Query live command metadata from the task's execution resources after presentation selection changes.
// Pattern: Read-only application projection over existing synchronized command sources.
// Usage: WebUI polls Commands at its status cadence; no subprocess or tool is started by this query.
// Related: internal/commands/teamrequest/activity.go, internal/tools/local/commands.go.
package server

import (
	"context"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"sort"
)

type CommandState struct {
	Task, Agent string
	Run         toolapi.CommandRun
}

func (s *Service) Commands(ctx context.Context, id string) ([]CommandState, error) {
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return nil, err
	}
	defer done()
	s.mu.Lock()
	r, err := s.currentRequest(id)
	var c *Conversation
	var release func()
	var team bool
	if r != nil {
		c = r.conversation
		if c != nil {
			team = c.TeamEnabled
			c, ctx, release, err = s.borrowReadLocked(ctx, c)
		}
	}
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, nil
	}
	defer release()
	var values []CommandState
	if team && c.Team != nil {
		for agent, runs := range c.Team.CommandActivity() {
			for _, run := range runs {
				values = append(values, CommandState{Task: id, Agent: agent, Run: run})
			}
		}
	} else if source, ok := c.Runtime.Tools.(interface{ CommandActivity() []toolapi.CommandRun }); ok {
		for _, run := range source.CommandActivity() {
			values = append(values, CommandState{Task: id, Run: run})
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Agent+values[i].Run.ID < values[j].Agent+values[j].Run.ID })
	return values, ctx.Err()
}
