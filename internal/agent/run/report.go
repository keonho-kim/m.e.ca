// Purpose: Render terminal execution facts without another model request.
// Pattern: Terminal run state transition.
// Usage: Run enters reporting after fulfilling the request, exhausted budgets or persistent no progress.
// Related: internal/agent/run/finish.go, internal/agent/run/completion_state.go, internal/agent/run/context.go.
package run

import (
	"context"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/response"
	"github.com/keonho-kim/v-code/internal/agent/work"
	"strings"
)

func (s *execution) report(ctx context.Context, reason string, stopped bool) (Result, error) {
	if err := ctx.Err(); err != nil {
		return s.result, err
	}
	if end := s.checkpoint.End; end != nil && end.Status == work.Denied {
		reason = end.Reason
	}
	s.checkpoint.Phase = "reporting"
	s.checkpoint.ReportReason = s.runner.Redact(reason)
	if len(s.checkpoint.ReportReason) > 256*1024*1024 {
		return s.result, fmt.Errorf("report reason exceeds resource bound")
	}
	if err := s.persist(ctx); err != nil {
		return s.result, err
	}
	text := s.reportFacts(stopped)
	finished, err := s.finish(ctx, text)
	if err != nil {
		return s.result, err
	}
	if !finished {
		return s.result, fmt.Errorf("final report was empty")
	}
	if stopped {
		s.checkpoint.Status = Stopped
	}
	return s.result, nil
}

func (s *execution) reportFacts(stopped bool) string {
	var out strings.Builder
	if stopped {
		out.WriteString("작업을 중단했습니다. 전체 완료로 처리하지 않았습니다.\n\n")
	} else {
		out.WriteString("요청한 작업을 마쳤습니다.\n\n")
	}
	if s.checkpoint.LastChange != "" {
		out.WriteString("- 마지막 성공한 변경: " + s.checkpoint.LastChange + "\n")
	}
	out.WriteString(s.checkpoint.Work.Facts().Describe())
	if s.checkpoint.Observation != "" {
		out.WriteString("- 마지막 관찰: " + excerpt(s.checkpoint.Observation, 768) + "\n")
	}
	reason := s.checkpoint.ReportReason
	if response.ToolMarkup(reason) {
		reason = "응답에 실행되지 않은 도구 형식이 포함됐습니다"
	}
	out.WriteString("- 종료 사유: " + excerpt(reason, 2048) + "\n")
	out.WriteString("- 미검증 사항: 기록된 실행 결과 외의 정확성·검증 통과 여부는 확인하지 않았습니다.")
	return out.String()
}
