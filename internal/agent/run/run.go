// Purpose: Coordinate preparation, conversation admission, decision work and cleanup for one request.
// Pattern: State Machine / Use Case.
// Usage: Invoked once per submitted user request; cancellation never replays effects.
// Related: internal/agent/run/start.go, internal/agent/run/conversation.go, internal/agent/run/loop.go.
package run

import (
	"context"

	"time"

	contextbuild "github.com/keonho-kim/meca/internal/agent/context"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/diagnostics"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/processmemory"
)

type execution struct {
	contextMemory       *processmemory.ContextLease
	replyMemory         *processmemory.ReplyLease
	projection          contextbuild.Projection
	historyLoaded       bool
	runner              Runner
	limits              Limits
	system              string
	result              Result
	checkpoint          runrecord.Checkpoint
	emit                func(runrecord.Event) error
	definitions         []toolapi.Definition
	usage               requestBudget
	statistics          runStatistics
	messages            []modelapi.Message
	skillReads          []modelapi.Message
	skillReadIndex      map[string]int
	skillReadKeys       []string
	skillReadRevisions  map[string]string
	skillFileVersions   map[string]string
	skillReadBytes      int
	explicitSkills      []string
	memoryContext       string
	capacityRecovery    bool
	commandRevision     uint64
	userSequence        int64
	instructionRevision uint64
}

func (r Runner) Run(ctx context.Context, input string, approve Approver, emit func(runrecord.Event) error) (result Result, err error) {
	started := time.Now()
	ctx = diagnostics.WithRedactor(ctx, r.Redact)
	if approve == nil || emit == nil {
		return result, faults.Errorf(diagcatalog.RunExecutionDependenciesAreRequired)
	}
	s, release, err := r.start(ctx, input, emit, started)
	if err != nil {
		return result, err
	}
	defer release()
	ctx = diagnostics.WithFields(ctx, "component", "agent", "session_id", r.Session, "run_id", s.checkpoint.ID)
	diagnostics.Log(ctx, diagcatalog.RunStarted)
	defer func() {
		if recover() != nil {
			err = faults.Panic()
		}
		err = s.close(ctx, err)
		result = s.result
		diagnostics.Log(ctx, diagcatalog.RunFinished, "status", string(result.Status), "duration_ms", time.Since(started).Milliseconds(), "error_code", diagnostics.Code(err))
	}()
	if err := s.startConversation(ctx, input); err != nil {
		return s.result, err
	}
	return s.result, s.drive(ctx, approve)
}
