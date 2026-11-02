// Purpose: Coordinate preparation, conversation admission, decision work and cleanup for one request.
// Pattern: State Machine / Use Case.
// Usage: Invoked once per submitted user request; cancellation never replays effects.
// Related: internal/agent/run/start.go, internal/agent/run/conversation.go, internal/agent/run/loop.go.
package run

import (
	"context"
	"fmt"
	"time"

	contextbuild "github.com/keonho-kim/meca/internal/agent/context"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type execution struct {
	projection       contextbuild.Projection
	historyLoaded    bool
	runner           Runner
	limits           Limits
	system           string
	result           Result
	checkpoint       runrecord.Checkpoint
	emit             func(runrecord.Event) error
	definitions      []toolapi.Definition
	usage            requestBudget
	statistics       runStatistics
	messages         []modelapi.Message
	memoryContext    string
	capacityRecovery bool
	commandRevision  uint64
}

func (r Runner) Run(ctx context.Context, input string, approve Approver, emit func(runrecord.Event) error) (result Result, err error) {
	started := time.Now()
	if approve == nil || emit == nil {
		return result, fmt.Errorf("execution dependencies are required")
	}
	s, release, err := r.start(ctx, input, emit, started)
	if err != nil {
		return result, err
	}
	defer release()
	defer func() {
		err = s.close(ctx, err)
		result = s.result
	}()
	if err := s.startConversation(ctx, input); err != nil {
		return s.result, err
	}
	return s.result, s.drive(ctx, approve)
}
