// Purpose: Publish the last peer's answer while departed peers finish resource cleanup.
// Pattern: Deterministic joined-lifetime regression test.
// Usage: go test -race ./internal/agent/team -run TestFinalAnswerDuringDeparture.
// Related: internal/agent/team/answer.go, internal/agent/team/worker.go.
package team

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/run"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
	"github.com/keonho-kim/meca/internal/agent/work"
)

func TestFinalAnswerDuringDeparture(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		departing, cleanup := make(chan struct{}), make(chan struct{})
		published := make(chan runrecord.Event, 1)
		done := make(chan error, 1)
		go func() {
			_, err := Run(ctx, Options{Store: &memoryRoom{}, MaxModelCalls: 10, MaxSeconds: 60,
				Emit: func(event runrecord.Event) error {
					if event.Final {
						published <- event
					}
					return nil
				},
				Execute: func(ctx context.Context, member identity.Member, _ string, peer *Peer) (run.Result, error) {
					if member.ID == "Hana" {
						defer func() { <-cleanup }()
						if _, err := peer.Inbox(ctx); err != nil {
							return run.Result{}, err
						}
						peer.BeginDecision()
						left, err := peer.Leave(ctx, "Contribution complete")
						close(departing)
						return run.Result{Departed: left}, err
					}
					<-departing
					if _, err := peer.Inbox(ctx); err != nil {
						return run.Result{}, err
					}
					peer.BeginDecision()
					left, err := peer.Leave(ctx, "Ready to report the final result")
					if err != nil {
						return run.Result{}, err
					}
					if left {
						t.Error("last peer departed instead of reporting")
					}
					return run.Result{Text: "Final checked answer"}, nil
				}}, "Complete the task")
			done <- err
		}()
		synctest.Wait()
		select {
		case event := <-published:
			if event.Agent != "Ren" || event.Text != "Final checked answer" {
				t.Errorf("unexpected final response: %+v", event)
			}
		default:
			t.Error("completed final answer withheld until departed peer cleanup")
		}
		select {
		case err := <-done:
			t.Fatalf("request returned before joining cleanup: %v", err)
		default:
		}
		close(cleanup)
		synctest.Wait()
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	})
}

func TestDepartedPeerCleanupFailureAfterFinalAnswer(t *testing.T) {
	want := errors.New("departed private session flush failed")
	departing, published := make(chan struct{}), make(chan struct{})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var saved runrecord.Event
	result, err := Run(ctx, Options{Store: &memoryRoom{}, MaxModelCalls: 10, MaxSeconds: 5,
		Emit: func(event runrecord.Event) error {
			if event.Final {
				close(published)
			}
			return nil
		},
		Record: func(_ context.Context, event runrecord.Event) error {
			if event.Kind == "run_checkpoint" {
				saved = event
			}
			return nil
		},
		Execute: func(ctx context.Context, member identity.Member, _ string, peer *Peer) (run.Result, error) {
			if member.ID == "Ren" {
				<-departing
			}
			if _, err := peer.Inbox(ctx); err != nil {
				return run.Result{}, err
			}
			peer.BeginDecision()
			if member.ID == "Ren" {
				return run.Result{Text: "Completed answer"}, nil
			}
			left, err := peer.Leave(ctx, "Contribution complete")
			close(departing)
			if err != nil {
				return run.Result{}, err
			}
			select {
			case <-published:
			case <-ctx.Done():
				return run.Result{}, ctx.Err()
			}
			return run.Result{Departed: left}, want
		}}, "Complete the request")
	if !errors.Is(err, want) || result.End.Status != work.Failed || result.Status != runrecord.Stopped {
		t.Fatalf("late cleanup failure hidden: result=%+v error=%v", result, err)
	}
	if saved.Checkpoint == nil || saved.Checkpoint.End.Status != work.Failed || result.Text != "Completed answer" {
		t.Fatal("final text or durable failure outcome lost", saved, result)
	}
}
