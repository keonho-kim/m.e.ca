// Purpose: Apply one tool outcome to change revisions, verification and progress state.
// Pattern: Observed state transition.
// Usage: action calls observeOutcome after durable result recording.
// Related: internal/agent/run/action.go, internal/agent/progress/files.go.
package run

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/agent/verification"
	"github.com/keonho-kim/v-code/internal/policy"
)

func (s *execution) observeOutcome(ctx context.Context, call toolapi.Call, content []byte, signature [32]byte, mutating bool, effect policy.Effect, result string, outcome toolapi.Result, toolErr error) error {
	s.checkpoint.Phase = "observation"
	s.checkpoint.Arguments = nil
	s.checkpoint.Observation = call.Name + " result: " + excerpt(s.runner.Redact(outcome.Summary), 512)
	if mutating && (outcome.Started || outcome.Failure != nil && outcome.Failure.Code == toolapi.StaleTarget) {
		if outcome.Started {
			s.mutations[signature] = true
		}
		s.changed = true
		s.checkpoint.ChangeRevision++
		s.checkpoint.Evidence = ""
	}
	if s.checkpoint.Checking && effect == policy.Command {
		if toolErr != nil {
			outcome.Failed = true
		}
		checks, err := verification.Observe(s.checkpoint.Checks, call, outcome, s.checkpoint.ChangeRevision)
		if err != nil {
			return err
		}
		s.checkpoint.Checks = checks
	}
	if outcome.Failed || toolErr != nil {
		s.checkpoint.Evidence = ""
		s.checkpoint.NoProgress++
		return s.persist(ctx)
	}
	if effect == policy.Read && outcome.Started && !s.changed {
		s.checkpoint.Evidence = excerpt(call.Name+" => observed read", 4000)
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
	}
	fingerprint := sha256.Sum256([]byte(fmt.Sprintf("%d:%s\n%s", s.checkpoint.ChangeRevision, call.Name+target, result)))
	if s.files.Repeated(outcome.Changes) || s.seen[fingerprint] {
		s.checkpoint.NoProgress++
		s.checkpoint.Observation += "\nThis call returned an already observed state without new evidence. Reuse the result and advance the user request; do not repeat the same unchanged query."
	} else {
		s.seen[fingerprint] = true
		s.evidenceRevision++
		s.checkpoint.NoProgress = 0
	}
	if s.checkpoint.Checking && len(verification.Pending(s.checkpoint.Checks, s.checkpoint.ChangeRevision)) == 0 && (!s.changed || effect == policy.Command) {
		s.checkpoint.Evidence = excerpt(call.Name+" "+string(content)+" => successful result", 4000)
	}
	return s.persist(ctx)
}
