// Purpose: Preserve request deduplication when a concurrent admission wins after an earlier ledger lookup.
// Pattern: Controlled durable-admission interleaving regression test.
// Usage: go test ./internal/commands/server -run TestStartRetryAcrossAdmission.
// Related: internal/commands/server/tasks.go, internal/commands/server/operations.go.
package server

import (
	"context"
	"testing"

	"github.com/keonho-kim/meca/internal/config/credentials"
)

type admissionLedger struct {
	lookup func() (Task, error)
	task   Task
}

func (l *admissionLedger) Get(ctx context.Context, _ string) (Task, error) {
	if err := ctx.Err(); err != nil {
		return Task{}, err
	}
	return l.lookup()
}
func (l *admissionLedger) Admit(context.Context, string, string, string) (Task, bool, error) {
	return l.task, true, nil
}
func (*admissionLedger) Finish(context.Context, Task) error { return nil }

func TestStartRetryAcrossAdmissionRace(t *testing.T) {
	s := detailFixture(&detailHistory{}, &credentials.Vault{})
	defer s.Close()
	session := s.State().Session.ID
	task := Task{ID: TaskID(session, "request-race"), Session: session, Status: "running"}
	reads := 0
	s.options.Ledger = &admissionLedger{task: task, lookup: func() (Task, error) {
		reads++
		if reads == 1 {
			// This lookup observed absence; another caller published the same key
			// and adopted its task before the first caller reserved idle state.
			s.mu.Lock()
			s.active = &request{task: task}
			s.mu.Unlock()
			return Task{}, ErrTaskNotFound
		}
		return task, nil
	}}
	got, err := s.Start(t.Context(), session, "request-race", "same prompt")
	if err != nil || got.ID != task.ID || got.Status != "running" {
		t.Fatal("a concurrent duplicate lost its original acknowledgement", got, err)
	}
}

func TestTaskStatusDistinguishesOwnedAdmissionFromRestart(t *testing.T) {
	s := detailFixture(&detailHistory{}, &credentials.Vault{})
	defer s.Close()
	stored := Task{ID: "admitted-task", Session: s.State().Session.ID, Status: "interrupted"}
	s.options.Ledger = &admissionLedger{lookup: func() (Task, error) { return stored, nil }}
	s.admitting = stored.ID
	value, err := s.TaskStatus(t.Context(), stored.ID)
	if err != nil || value.Status != "accepted" {
		t.Fatal("owned admission was mistaken for a restarted request", value, err)
	}
	s.admitting = ""
	value, err = s.TaskStatus(t.Context(), stored.ID)
	if err != nil || value.Status != "interrupted" {
		t.Fatal("unowned admission did not retain restart safety", value, err)
	}
}
