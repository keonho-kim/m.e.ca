// Purpose: Render terminal execution facts without another model request.
// Pattern: Terminal run state transition.
// Usage: Run reports policy and resource stops without another model generation.
// Related: internal/agent/run/finish.go, internal/agent/run/observations.go, internal/agent/run/context.go.
package run

import (
	"context"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"strings"
)

func (s *execution) stopReport(ctx context.Context, reason string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.checkpoint.Phase = "reporting"
	s.checkpoint.ReportReason = s.runner.Redact(reason)
	if err := s.persist(ctx); err != nil {
		return err
	}
	if err := s.finish(ctx, modelapi.Response{Text: s.stopFacts()}); err != nil {
		return err
	}
	s.checkpoint.Status = Stopped
	return nil
}

func (s *execution) stopFacts() string {
	var out strings.Builder
	out.WriteString("작업을 중단했습니다. 전체 완료로 처리하지 않았습니다.\n\n")
	if s.checkpoint.LastChange != "" {
		out.WriteString("- 마지막 성공한 변경: " + s.checkpoint.LastChange + "\n")
	}
	out.WriteString(s.checkpoint.Work.Describe())
	if s.checkpoint.Observation != "" {
		out.WriteString("- 마지막 관찰: " + excerpt(s.checkpoint.Observation, 768) + "\n")
	}
	reason := s.checkpoint.ReportReason
	out.WriteString("- 종료 사유: " + excerpt(reason, 2048) + "\n")
	out.WriteString("- 미검증 사항: 기록된 실행 결과 외의 정확성·검증 통과 여부는 확인하지 않았습니다.")
	return out.String()
}
