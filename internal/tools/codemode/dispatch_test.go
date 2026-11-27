// Purpose: Verify individual completions resume code while sibling calls remain pending.
// Pattern: Deterministic host delivery scripts.
// Usage: go test ./internal/tools/codemode.
// Related: internal/tools/codemode/engine.go, internal/tools/codemode/contract.go.
package codemode

import (
	"context"
	"encoding/json"
	"testing"
)

func replyScript(reply func(context.Context, []Call) ([]Reply, error)) Dispatch {
	return func(ctx context.Context, calls []Call, deliver Deliver) error {
		for len(calls) != 0 {
			replies, err := reply(ctx, calls)
			if err != nil {
				return err
			}
			var pending []Call
			for i, call := range calls {
				next, err := deliver(call.ID, replies[i])
				if err != nil {
					return err
				}
				pending = append(pending, next...)
			}
			calls = pending
		}
		return nil
	}
}

func TestFastResultStartsContinuationBeforeSlowSibling(t *testing.T) {
	result, err := Run(t.Context(), `
		const slow = tools.read({id: 1});
		const fast = tools.read({id: 2});
		const value = await fast;
		await tools.read({id: value});
		return await slow;
	`, []string{"read"}, func(_ context.Context, calls []Call, deliver Deliver) error {
		if len(calls) != 2 {
			t.Fatalf("initial calls=%+v", calls)
		}
		next, err := deliver(calls[1].ID, Reply{Value: json.RawMessage(`3`)})
		if err != nil || len(next) != 1 || string(next[0].Arguments) != `{"id":3}` {
			t.Fatalf("fast continuation=%+v err=%v", next, err)
		}
		if _, err := deliver(next[0].ID, Reply{Value: json.RawMessage(`true`)}); err != nil {
			return err
		}
		_, err = deliver(calls[0].ID, Reply{Value: json.RawMessage(`1`)})
		return err
	})
	if err != nil || string(result.Value) != `1` || result.Calls != 3 {
		t.Fatalf("%+v err=%v", result, err)
	}
}

func TestPendingLimitIncludesAlreadyDispatchedCalls(t *testing.T) {
	_, err := Run(t.Context(), `
		const pending = Array.from({length: 32}, () => tools.read({}));
		await pending[0];
		pending.push(tools.read({}), tools.read({}));
		return await Promise.all(pending);
	`, []string{"read"}, func(_ context.Context, calls []Call, deliver Deliver) error {
		_, err := deliver(calls[0].ID, Reply{Value: json.RawMessage(`true`)})
		return err
	})
	if err == nil {
		t.Fatal("dispatched calls escaped the pending limit")
	}
}
