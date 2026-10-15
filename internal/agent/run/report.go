// Purpose: End work through one tool-free report, with a factual fallback when generation is unavailable.
// Pattern: Terminal run state transition.
// Usage: Run enters reporting after fulfilling the request, exhausted budgets or persistent no progress.
// Related: internal/agent/run/finish.go, internal/agent/run/completion_state.go, internal/agent/run/context.go.
package run

import (
	"context"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/response"
	"strings"
	"unicode/utf8"

	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
)

func (s *execution) report(ctx context.Context, reason string, stopped bool) (Result, error) {
	if err := ctx.Err(); err != nil {
		return s.result, err
	}
	s.checkpoint.Phase = "reporting"
	s.checkpoint.ReportReason = s.runner.Redact(reason)
	if len(s.checkpoint.ReportReason) > 256*1024*1024 {
		return s.result, fmt.Errorf("report reason exceeds resource bound")
	}
	stopped = stopped || s.checkpoint.PendingReview
	if err := s.persist(ctx); err != nil {
		return s.result, err
	}
	text := ""
	unavailable := ""
	// The final decision slot is reserved at the main loop boundary. If reviews,
	// retries or elapsed time used the remaining budget, report recorded facts.
	if !stopped && s.usage.ModelCalls < s.limits.MaxModelCalls && s.checkTimeBudget(ctx) == nil {
		role := inference.Agent

		reply, err := s.askResponse(ctx, "Work has ended. Produce the final report only; no more tool calls or speculative improvements. Completion evidence links model claims to actual successful operations, not independent verification. Distinguish completed changes, actual checks, and incomplete or unverified items. If stopped, state the concrete remaining work. Use the user's language. Never claim overall success merely because an operation succeeded.", role, modelapi.MaxResponseBytes)
		if ctx.Err() != nil {
			return s.result, ctx.Err()
		}
		if errors.Is(err, errRuntimeRecord) {
			return s.result, err
		}
		if err != nil {
			unavailable = s.runner.Redact(err.Error())
		} else {
			text = strings.TrimSpace(reply.Text)
		}
	}
	if response.ToolMarkup(text) {
		unavailable = "model returned non-executed tool syntax instead of a report"
		text = ""
	}
	if text == "" || !utf8.ValidString(text) {
		stopped = true
		text = s.reportFallback(unavailable)
	}
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

func (s *execution) reportFallback(unavailable string) string {
	var out strings.Builder
	out.WriteString("작업을 중단했습니다. 전체 완료로 처리하지 않았습니다.\n\n")
	if s.checkpoint.LastChange != "" {
		out.WriteString("- 마지막 성공한 변경: " + s.checkpoint.LastChange + "\n")
	}
	for _, e := range s.checkpoint.Work.Receipts {
		label := "성공한 실행"
		if !e.Success {
			label = "실패 또는 미실행"
		} else if e.Stale {
			label = "이전 성공(현재 상태 미확인)"
		}
		fmt.Fprintf(&out, "- %s: %s (%s)\n", label, strings.SplitN(e.Output, "\n", 2)[0], e.Tool)
	}
	if s.checkpoint.Observation != "" {
		out.WriteString("- 마지막 관찰: " + excerpt(s.checkpoint.Observation, 768) + "\n")
	}
	reason := s.checkpoint.ReportReason
	if response.ToolMarkup(reason) {
		reason = "심사 응답에 실행되지 않은 도구 형식이 포함됐습니다"
	}
	out.WriteString("- 종료 사유: " + reason + "\n")
	if s.checkpoint.Objective != "" {
		out.WriteString("- 요청 기준(전체 충족을 보증하지 않음): " + excerpt(s.checkpoint.Objective, 2048) + "\n")
	}
	if unavailable != "" {
		out.WriteString("- 최종 보고 생성 실패: " + excerpt(unavailable, 512) + "\n")
	}
	out.WriteString("- 미검증 사항: 기록된 실행 결과 외의 정확성·검증 통과 여부는 확인하지 않았습니다.")
	return out.String()
}
