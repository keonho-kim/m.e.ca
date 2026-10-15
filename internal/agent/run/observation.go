// Purpose: Apply actual tool outcomes to change revisions and progress state.
// Pattern: Observed state transition.
// Usage: action calls observeOutcome after durable result recording.
// Related: internal/agent/run/action.go, internal/agent/progress/files.go.
package run

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
)

func (s *execution) observeOutcome(ctx context.Context, call toolapi.Call, content []byte, signature [32]byte, mutating bool, effect policy.Effect, result string, outcome toolapi.Result, toolErr error) error {
	s.checkpoint.Phase = "observation"
	s.checkpoint.Arguments = nil
	s.checkpoint.Observation = call.Name + " result: " + excerpt(s.runner.Redact(outcome.Summary), 512)
	if mutating && (outcome.Started || outcome.Failure != nil && outcome.Failure.Code == toolapi.StaleTarget) {
		if outcome.Started && effect == policy.Change {
			s.mutations[signature] = true
		}
		s.checkpoint.ChangeRevision++
		// Real file changes make earlier reads/checks stale. Merely running another
		// command must not erase the evidence that an observation already repeated.
		if effect == policy.Change {
			clear(s.seen)
		}
		if outcome.Started && effect == policy.Command {
			s.commands[signature] = s.checkpoint.ChangeRevision
		}
	}
	if outcome.Failed || toolErr != nil {
		if outcome.Failure != nil && outcome.Failure.Code == toolapi.ScopeUnavailable {
			s.checkpoint.Blocker = excerpt(s.runner.Redact(outcome.Failure.Detail), 3500)
			s.checkpoint.Observation = "Command was not started: " + s.checkpoint.Blocker + ". Resolve the runtime setup; do not widen Scope or claim verification passed."
		}
		s.checkpoint.NoProgress++
		return s.persist(ctx)
	}
	if effect == policy.Change && outcome.Started {
		s.checkpoint.LastChange = excerpt(call.Name+": "+s.runner.Redact(outcome.Summary), 768)
	}
	if effect == policy.Command && outcome.Started {
		s.checkpoint.Blocker = ""
	}
	if mutating && outcome.Started && outcome.ReviewSatisfied {
		s.checkpoint.PendingReview = false
	}
	target := outcome.Target
	if target == "" {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(content, &fields); err != nil {
			return err
		}
		target = string(fields["path"])
		if target == "" {
			target = string(fields["command"])
		}
	}
	fingerprint := sha256.Sum256([]byte(call.Name + "\n" + target + "\n" + result))
	cycle := s.files.Repeated(outcome.Changes)
	if cycle || s.seen[fingerprint] {
		s.checkpoint.NoProgress++
		s.checkpoint.Observation += "\nThis call returned an already observed state without new evidence. Reuse the result and advance the user request; do not repeat the same unchanged query."
	} else {
		s.seen[fingerprint] = true
		s.evidenceRevision++
		s.checkpoint.NoProgress = 0
	}
	if cycle {
		s.checkpoint.Observation += "\nFile contents returned to a previously observed state. Stop rewriting the same result; inspect the concrete failed requirement and verify before another edit."
		s.checkpoint.NoProgress = max(s.checkpoint.NoProgress, s.limits.StallIterations)
		return s.persist(ctx)
	}
	return s.persist(ctx)
}
