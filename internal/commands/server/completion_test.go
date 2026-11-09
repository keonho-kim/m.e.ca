// Purpose: Distinguish acknowledged durable completion from uncertain or mismatched readback.
// Pattern: Controlled persistence failure tests.
// Usage: go test ./internal/commands/server -run TestCompletionReadback.
// Related: internal/commands/server/completion.go, internal/commands/server/admission_test.go.
package server

import (
	"context"
	"errors"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"testing"
)

func TestCompletionReadbackSurvivesCancellationWithoutReplaying(t *testing.T) {
	want := Task{ID: "task", Session: "session", Status: "answered", Persistence: PersistenceConfirmed}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	reads := 0
	s := &Service{options: Options{Ledger: &admissionLedger{lookup: func() (Task, error) { reads++; return want, nil }}}}
	writeErr := errors.New("commit acknowledgement lost")
	if err := s.confirmCompletion(ctx, want, nil, writeErr); err != nil || reads != 1 {
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
		if err := s.confirmCompletion(ctx, want, nil, writeErr); !errors.Is(err, writeErr) {
			t.Fatal(field, "mismatched readback erased uncertainty", err)
		}
	}
}

type admissionResponseStore struct {
	value   Response
	persist bool
	saves   int
}

func (r *admissionResponseStore) ReadResponse(context.Context, string) (Response, error) {
	if r.value.ID == "" {
		return Response{}, fault.New(catalog.NotFound, catalog.ServerResponseWasNotFound)
	}
	return r.value, nil
}
func (r *admissionResponseStore) ReserveResponse(_ context.Context, value Response) (Response, bool, error) {
	r.value = value
	return value, false, nil
}
func (r *admissionResponseStore) SaveResponse(_ context.Context, value Response) error {
	r.saves++
	if r.persist {
		r.value = value
	}
	return errors.New("completion acknowledgement unavailable")
}

func TestAdmissionFailureConfirmsItsTerminalWrite(t *testing.T) {
	for _, persisted := range []bool{false, true} {
		t.Run(map[bool]string{false: "uncertain", true: "confirmed"}[persisted], func(t *testing.T) {
			s := detailFixture(&detailHistory{}, &credentials.Vault{})
			t.Cleanup(func() { _ = s.Close() })
			store := &admissionResponseStore{persist: persisted}
			s.options.Responses = store
			s.options.ReadOutput = func(context.Context, string, int64, int64, int, func(int) error) (string, error) { return "", nil }
			s.options.Open = func(context.Context, string, bool, config.Config) (*Conversation, error) {
				return nil, errors.New("session source unavailable")
			}
			value, err := s.Submit(t.Context(), Submission{Kind: "response", Key: "request", Messages: []InputMessage{{Role: "user", Text: "input"}}})
			if err == nil || store.saves != 1 {
				t.Fatal("failed admission was replayed or reported successful", store.saves, err)
			}
			if persisted {
				if s.requireIdle() != nil || value.Response.Code != catalog.AdmissionFailed {
					t.Fatal("confirmed failure remained uncertain", value, err)
				}
			} else if fault.Code(s.requireIdle()) != catalog.CompletionNotSaved || value.Response.Code != catalog.CompletionNotSaved {
				t.Fatal("unconfirmed admission did not block further work", value, err)
			}
		})
	}
}
