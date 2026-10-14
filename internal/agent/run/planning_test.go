// Purpose: Verify ordinary Plan-mode conversation stays read-only without requiring or replacing a completed plan.
// Pattern: Planning use-case regression test.
// Usage: go test ./internal/agent/run -run TestPlanConversation.
// Related: internal/agent/run/planning.go, internal/agent/run/run.go, internal/agent/run/response.go.
package run

import (
	"context"
	"testing"
	"time"

	planning "github.com/keonho-kim/v-code/internal/agent/plan"
)

type planRecorder struct{ plans []planning.Plan }

func (s *planRecorder) SavePlan(_ context.Context, p planning.Plan) error {
	s.plans = append(s.plans, p)
	return nil
}

func TestPlanConversationDoesNotRequireProposal(t *testing.T) {
	for _, reply := range []string{"DONE\nHello!", "DONE\n현재 요구사항부터 확인하겠습니다.", "WAIT\n어떤 기능을 계획할까요?", "DONE\n{\"example\":true}"} {
		t.Run(reply, func(t *testing.T) {
			g, e, j := &generator{replies: []string{reply}}, &executor{}, &journal{}
			r := fixture(g, e, j)
			store := &planRecorder{}
			previous, err := planning.NewPlan("p", "s", "existing objective", []string{"read first"}, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			updated, err := r.DraftPlan(context.Background(), "hello", &previous, store, approve, emit)
			if err != nil || updated != nil || len(store.plans) != 0 || e.executed != 0 {
				t.Fatalf("ordinary reply became a plan or error: %+v %v", updated, err)
			}
			if previous.Revision != 1 || previous.Status != planning.AwaitingApproval {
				t.Fatal("ordinary conversation changed the pending Plan")
			}
		})
	}
}

func TestPlanConversationPublishesOnlyCompletePlan(t *testing.T) {
	g, e, j := &generator{replies: []string{`DONE
{"objective":"inspect project","steps":["Read the entry point","Propose a change"]}`}}, &executor{}, &journal{}
	r := fixture(g, e, j)
	store := &planRecorder{}
	updated, err := r.DraftPlan(context.Background(), "make a plan", nil, store, approve, emit)
	if err != nil || updated == nil || updated.Status != planning.AwaitingApproval || len(store.plans) != 1 || e.executed != 0 {
		t.Fatalf("completed plan: %+v %v", updated, err)
	}
}

func TestPlanConversationCanAnswerAfterInspection(t *testing.T) {
	g, e, j := &generator{replies: []string{`{"tool":"read_file"}`, "README.md", "DONE\n이 프로젝트는 CLI 프로그램입니다."}}, &executor{}, &journal{}
	r := fixture(g, e, j)
	result, err := r.Run(context.Background(), "프로젝트를 설명해줘", true, approve, emit)
	if err != nil || result.Status != Answered || e.executed != 1 {
		t.Fatalf("read-only discussion: %+v %v", result, err)
	}
}
