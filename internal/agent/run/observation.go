// Purpose: Apply actual tool outcomes to change revisions and progress state.
// Pattern: Observed state transition.
// Usage: action calls observeOutcome after durable result recording.
// Related: internal/agent/run/action.go, internal/agent/progress/files.go.
package run

import (
	"context"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"maps"
)

func (s *execution) observeOutcome(ctx context.Context, call toolapi.Call, content []byte, signature [32]byte, mutating bool, effect policy.Effect, result string, outcome toolapi.Result, toolErr error) error {
	s.checkpoint.Phase = "observation"
	s.checkpoint.Arguments = nil
	s.checkpoint.Observation = call.Name + " result: " + excerpt(s.runner.Redact(outcome.Summary), 512)
	if mutating && (outcome.Started || outcome.Failure != nil && outcome.Failure.Code == toolapi.StaleTarget) {
		if outcome.Started && effect == policy.Change {
			if s.mutations == nil {
				s.mutations = map[[32]byte]mutationReceipt{}
			}
			s.mutations[signature] = mutationReceipt{call.ID, maps.Clone(outcome.Versions), !outcome.Failed && toolErr == nil}
		}
		s.checkpoint.ChangeRevision++
		if outcome.Started && effect == policy.Command {
			s.commands[signature] = s.checkpoint.ChangeRevision
		}
	}
	if outcome.Failed || toolErr != nil {
		if outcome.Failure != nil && outcome.Failure.Code == toolapi.ScopeUnavailable {
			s.checkpoint.Blocker = excerpt(s.runner.Redact(outcome.Failure.Detail), 3500)
			s.checkpoint.Observation = "Command was not started: " + s.checkpoint.Blocker
		}
		if outcome.Failure == nil {
			outcome.Failure = toolapi.FailureOf(toolErr)
		}
		if s.toolAdvanced(call, outcome, effect, result) {
			s.round.advanced = true
		}
		return s.persist(ctx)
	}
	if effect == policy.Change && outcome.Started {
		s.checkpoint.LastChange = excerpt(call.Name+": "+s.runner.Redact(outcome.Summary), 768)
	}
	if effect == policy.Command && outcome.Started {
		s.checkpoint.Blocker = ""
	}
	if effect == policy.Change && !changedFiles(outcome) {
		return s.persist(ctx)
	}
	cycle := s.files.Repeated(outcome.Changes)
	if !cycle && s.toolAdvanced(call, outcome, effect, result) {
		s.round.advanced = true
	} else {
		s.checkpoint.Observation += "\nThis call returned an already observed state. Reuse it and address one concrete remaining outcome."
	}

	if cycle {
		s.checkpoint.Observation += "\nFile contents returned to a previously observed state. Stop rewriting the same result; inspect the concrete failed requirement and verify before another edit."
		s.round.cycle = true
		return s.persist(ctx)
	}
	return s.persist(ctx)
}

func changedFiles(outcome toolapi.Result) bool {
	for _, c := range outcome.Changes {
		if c.BeforeDigest != c.AfterDigest || c.Destination != "" {
			return true
		}
	}
	return false
}
