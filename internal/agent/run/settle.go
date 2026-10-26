// Purpose: Settle every run exit into typed outcomes and observed effects before releasing request resources.
// Pattern: Terminal state transition.
// Usage: Runner.Run defers settle after establishing its checkpoint.
// Related: internal/agent/run/report.go, internal/agent/session/request.go.
package run

import (
	"context"
	"errors"
	"time"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/work"
)

func (s *execution) settle(ctx context.Context, runErr error) error {
	if runErr != nil {
		end := work.End{Status: work.Failed, Reason: s.runner.Redact(runErr.Error())}
		s.checkpoint.Status = runrecord.Stopped
		switch {
		case errors.Is(runErr, errModelBudget), errors.Is(runErr, errTimeBudget):
			end.Status = work.Exhausted
		case errors.Is(runErr, errApprovalDenied):
			end.Status = work.Denied
		case errors.Is(runErr, context.Canceled), errors.Is(runErr, context.DeadlineExceeded):
			end.Status = work.Interrupted
			s.checkpoint.Status = runrecord.Interrupted
		}
		s.checkpoint.End = &end
		s.checkpoint.ReportReason = end.Reason
	}
	if s.checkpoint.End == nil {
		status := work.Failed
		if s.checkpoint.Status == runrecord.Answered {
			status = work.Completed
		}
		s.checkpoint.End = &work.End{Status: status, Reason: s.checkpoint.ReportReason}
	}
	s.checkpoint.Phase = "finished"
	s.result.Status, s.result.End = s.checkpoint.Status, *s.checkpoint.End
	s.result.Facts = s.checkpoint.Work
	if runErr != nil {
		s.result.Text = "작업을 중단했습니다.\n" + s.result.Facts.Describe() + "- 종료 사유: " + s.checkpoint.End.Reason
	}
	if err := s.result.Facts.Check(); err != nil {
		runErr = errors.Join(runErr, err)
	}
	flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return errors.Join(runErr, s.persist(flush))
}

func (s *execution) stopBudget(ctx context.Context, reason string) error {
	s.checkpoint.End = &work.End{Status: work.Exhausted, Reason: reason}
	return s.stopReport(ctx, reason)
}
