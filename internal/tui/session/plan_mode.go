// Purpose: Toggle Plan presentation after persisted state transitions and await active planner cancellation.
// Pattern: Presentation use case coordination.
// Usage: !plan calls togglePlan; worker completion finishes a deferred leavePlan.
// Related: internal/agent/run/plan_approval.go, internal/tui/session/worker_result.go.
package session

import (
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/run"
)

func (a *app) togglePlan() error {
	if a.planMode {
		if a.busy {
			a.planStopping = true
			a.taskCancel()
			a.status = "계획 작업을 중단하고 Plan 모드를 종료합니다"
			return nil
		}
		return a.leavePlan()
	}
	p, err := run.BeginPlan(a.ctx, a.opts.Store, a.opts.Session.ID)
	if err != nil {
		return err
	}
	a.plan, a.planMode = &p, true
	a.status = "Plan 모드: 읽기·검색만 허용됩니다"
	return nil
}

func (a *app) leavePlan() error {
	if a.plan == nil {
		return fmt.Errorf("active Plan is missing")
	}
	p, err := run.CancelPlan(a.ctx, a.opts.Store, *a.plan)
	if err != nil {
		return err
	}
	a.plan, a.planMode = &p, false
	a.page, a.choices, a.choice = "chat", nil, 0
	a.status = "Plan 모드를 종료했습니다. 계획은 실행하지 않았습니다"
	return nil
}
