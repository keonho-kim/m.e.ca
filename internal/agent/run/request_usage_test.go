// Purpose: Verify request accounting uses final real totals, fallback bytes and separate retry attempts.
// Pattern: Model boundary behavior tests.
// Usage: go test ./internal/agent/run -run TestRequestAccounting.
// Related: internal/agent/run/request_usage.go, internal/agent/run/usage.go.
package run

import (
	"context"
	"errors"
	"github.com/keonho-kim/meca/internal/agent/compact"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"testing"
	"time"
)

type usageRecords struct {
	capacityStore
	counts []compact.TokenCount
}

func (r *usageRecords) AddContextTokens(_ context.Context, _ string, c compact.TokenCount) error {
	r.counts = append(r.counts, c)
	return nil
}

type attemptScript struct {
	events []modelapi.Event
	err    error
}

func (g attemptScript) Generate(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
	for _, e := range g.events {
		if err := emit(e); err != nil {
			return err
		}
	}
	return g.err
}

func TestRequestAccounting(t *testing.T) {
	for _, tc := range []struct {
		name      string
		events    []modelapi.Event
		err       error
		account   bool
		want      []int64
		estimated bool
	}{
		{"latest total", []modelapi.Event{{Usage: &modelapi.TokenUsage{Input: 100, Output: 2}}, {Usage: &modelapi.TokenUsage{Input: 100, Output: 5}, Done: true}}, nil, true, []int64{105}, false},
		{"reported zero", []modelapi.Event{{Delta: "hello", Usage: &modelapi.TokenUsage{}, Done: true}}, nil, true, []int64{0}, false},
		{"cancelled", []modelapi.Event{{Usage: &modelapi.TokenUsage{Input: 100, Output: 3}}}, context.Canceled, true, []int64{103}, false},
		{"retry", []modelapi.Event{{Usage: &modelapi.TokenUsage{Input: 100, Output: 3}}, {Retry: &modelapi.Retry{Attempt: 2, MaxAttempts: 3}}, {Usage: &modelapi.TokenUsage{Input: 110, Output: 5}, Done: true}}, nil, true, []int64{103, 115}, false},
		{"no usage", []modelapi.Event{{Delta: "응답", Thinking: "추론", Done: true}}, nil, true, []int64{-1}, true},
		{"compaction excluded", []modelapi.Event{{Usage: &modelapi.TokenUsage{Input: 100, Output: 5}, Done: true}}, nil, false, nil, false},
		{"request failed before response", nil, errors.New("connection failed"), true, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &usageRecords{}
			runner := fixture(&generator{}, &executor{}, &store.journal)
			runner.Generator = attemptScript{events: tc.events, err: tc.err}
			runner.Memory = &compact.Controller{Repository: store, Session: "s"}
			s := execution{runner: runner, limits: DefaultLimits(), usage: &requestBudget{deadline: time.Now().Add(time.Minute)}, emit: emit}
			err := s.generate(context.Background(), modelapi.Request{Messages: []modelapi.Message{{Role: "user", Content: "request"}}}, tc.account, func(modelapi.Event) error { return nil })
			if tc.err != nil && !errors.Is(err, tc.err) || tc.err == nil && err != nil {
				t.Fatal(err)
			}
			if len(store.counts) != len(tc.want) {
				t.Fatalf("counts=%+v", store.counts)
			}
			for i, want := range tc.want {
				got := store.counts[i]
				if got.Estimated != tc.estimated || got.Kind != "request" || want >= 0 && got.Amount != want || want < 0 && got.Amount <= 0 {
					t.Fatalf("count=%+v want=%d", got, want)
				}
			}
		})
	}
}
