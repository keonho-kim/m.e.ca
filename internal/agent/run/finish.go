// Purpose: End a model turn using observed verification state rather than text control words.
// Pattern: Execution completion transition.
// Usage: The single loop calls finish for a response without native calls.
// Related: internal/agent/run/run.go, internal/agent/verification/state.go.
package run

import (
	"context"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/verification"
	"strings"
	"unicode/utf8"
)

func (s *execution) finish(ctx context.Context, text string) (bool, error) {
	text = strings.TrimSpace(text)
	if text == "" || !utf8.ValidString(text) {
		s.checkpoint.NoProgress++
		s.checkpoint.Observation = "Return a nonempty answer or one native call."
		return false, nil
	}
	pending := s.checkpoint.PendingReview || len(verification.Pending(s.checkpoint.Checks, s.checkpoint.ChangeRevision)) > 0 || s.changed && s.checkpoint.Evidence == ""
	if pending && !s.planning {
		if !s.finishCorrection {
			s.finishCorrection = true
			s.checkpoint.Observation = "Required verification/review is still unresolved. After file changes, execute an appropriate command with purpose=check; a read alone is not a completed check. For a text artifact, compare the actual bytes with the requested contents. If blocked, explain why the check cannot run."
			return false, s.persist(ctx)
		}
		s.checkpoint.Status = Unverified
		if err := s.save(ctx, modelapi.Message{Role: "system", Content: "Unverified model completion (not accepted as success): " + text}); err != nil {
			return false, err
		}
		text = "작업을 검증 완료로 처리하지 못했습니다."
		if s.checkpoint.PendingReview && s.checkpoint.Conclave != nil {
			text += "\n- 심사: " + s.checkpoint.Conclave.Feedback()
		}
		for index, check := range verification.Pending(s.checkpoint.Checks, s.checkpoint.ChangeRevision) {
			if index >= 3 {
				text += "\n- 추가 미해결 검사는 대화 기록에 보존했습니다."
				break
			}
			text += "\n- " + check.Target + ": " + check.Status + " — " + check.Detail
		}
		if s.changed && s.checkpoint.Evidence == "" {
			text += "\n- 마지막 변경 이후의 성공한 검증이 필요합니다."
		}
	} else {
		s.checkpoint.Status = Answered
		if !s.planning && s.changed && s.checkpoint.Evidence != "" {
			s.checkpoint.Status = Verified
		}
	}
	s.result.Text = text
	if err := s.save(ctx, modelapi.Message{Role: "assistant", Content: text}); err != nil {
		return false, err
	}
	return true, s.emit(Event{Kind: "response", Text: text})
}
