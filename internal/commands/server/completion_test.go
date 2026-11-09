// Purpose: Distinguish acknowledged durable completion from uncertain or mismatched readback.
// Pattern: Controlled persistence failure tests.
// Usage: go test ./internal/commands/server -run TestCompletionReadback.
// Related: internal/commands/server/completion.go, internal/commands/server/admission_test.go.
package server

import (
	"context"
	"errors"
	"testing"
)

func TestCompletionReadbackSurvivesCancellationWithoutReplaying(t *testing.T) {
	want := Task{ID: "task", Session: "session", Status: "answered", Persistence: PersistenceConfirmed}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	reads := 0
	s := &Service{options: Options{Ledger: &admissionLedger{lookup: func() (Task, error) { reads++; return want, nil }}}}
	writeErr := errors.New("commit acknowledgement lost")
	if err := s.confirmCompletion(ctx, want, writeErr); err != nil || reads != 1 {
		t.Fatal("durable completion was not confirmed by one read", reads, err)
	}
	for _, field := range []string{"status", "code", "persistence"} {
		actual := want
		switch field {
		case "status":
			actual.Status = "interrupted"
		case "code":
			actual.Code = "mismatched-code"
		case "persistence":
			actual.Persistence = PersistenceUncertain
		}
		s.options.Ledger = &admissionLedger{lookup: func() (Task, error) { return actual, nil }}
		if err := s.confirmCompletion(ctx, want, writeErr); !errors.Is(err, writeErr) {
			t.Fatal(field, "mismatched readback erased uncertainty", err)
		}
	}
}
