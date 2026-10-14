// Purpose: Apply one tool outcome to change revisions, verification and progress state.
// Pattern: Observed state transition.
// Usage: action calls observeOutcome after durable result recording.
// Related: internal/agent/run/action.go, internal/agent/progress/files.go.
package run

import (
	"context"
	"crypto/sha256"
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/agent/verification"
	"github.com/keonho-kim/v-code/internal/policy"
)

func (s *task) observeOutcome(ctx context.Context, call toolapi.Call, content []byte, signature [32]byte, mutating bool, effect policy.Effect, result string, outcome toolapi.Result, toolErr error) error {
	s.checkpoint.Phase = "observation"
	s.checkpoint.Arguments = nil
	s.checkpoint.Observation = excerpt(call.Name+" result:\n"+s.runner.Redact(result), 8192)
	if mutating && (outcome.Started || outcome.Failure != nil && outcome.Failure.Code == toolapi.StaleTarget) {
		if outcome.Started {
			s.mutations[signature] = true
		}
		s.changed = true
		s.checkpoint.ChangeRevision++
		s.checkpoint.Evidence = ""
	}
	if s.checkpoint.Checking {
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
		if outcome.Failure != nil && (outcome.Failure.Code == reviewRejected || outcome.Failure.Code == reviewFailed) {
			s.checkpoint.NoProgress = 0
		} else {
			s.checkpoint.NoProgress++
		}
		return s.persist(ctx)
	}
	if mutating && outcome.Started && outcome.ReviewSatisfied {
		s.checkpoint.PendingReview = false
	}
	fingerprint := sha256.Sum256([]byte(fmt.Sprintf("%d:%x\n%s", s.checkpoint.ChangeRevision, signature, result)))
	if s.files.Repeated(outcome.Changes) || s.seen[fingerprint] {
		s.checkpoint.NoProgress++
	} else {
		s.seen[fingerprint] = true
		s.checkpoint.NoProgress = 0
	}
	if s.checkpoint.Checking && len(verification.Pending(s.checkpoint.Checks, s.checkpoint.ChangeRevision)) == 0 && (!s.changed || effect == policy.Command) {
		s.checkpoint.Evidence = excerpt(call.Name+" "+string(content)+" => successful result", 4000)
	}
	return s.persist(ctx)
}
