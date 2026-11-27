// Purpose: Verify composed JavaScript and TypeScript calls, isolation and bounded execution.
// Pattern: Functional tests with deterministic host tool callbacks.
// Usage: go test ./internal/tools/codemode.
// Related: internal/tools/codemode/engine.go, internal/tools/codemode/transform.go.
package codemode

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestTypeScriptComposesAndFiltersToolResults(t *testing.T) {
	var seen []string
	result, err := Run(t.Context(), `
		const items: number[] = await tools.list({});
		const selected: number[] = [];
		for (const id of items) {
			const item = await tools.read({id});
			if (item.active) selected.push(id);
		}
		return selected;
	`, []string{"list", "read"}, replyScript(func(_ context.Context, calls []Call) ([]Reply, error) {
		if len(calls) != 1 {
			t.Fatalf("dependent calls were batched: %+v", calls)
		}
		c := calls[0]
		seen = append(seen, c.Name)
		if c.Name == "list" {
			return []Reply{{Value: json.RawMessage(`[1,2,3]`)}}, nil
		}
		return []Reply{{Value: json.RawMessage(`{"active":` + map[bool]string{true: "true", false: "false"}[string(c.Arguments) == `{"id":2}`] + `}`)}}, nil
	}))
	if err != nil || string(result.Value) != `[2]` || strings.Join(seen, ",") != "list,read,read,read" {
		t.Fatalf("result=%s calls=%v err=%v", result.Value, seen, err)
	}
}

func TestPromisesBatchIndependentCallsAndFreezeArguments(t *testing.T) {
	result, err := Run(t.Context(), `
		const args = {id: 1};
		const a = tools.read(args); args.id = 9;
		const b = tools.read({id: 2});
		return await Promise.all([a, b]);
	`, []string{"read"}, replyScript(func(_ context.Context, calls []Call) ([]Reply, error) {
		if len(calls) != 2 || string(calls[0].Arguments) != `{"id":1}` || string(calls[1].Arguments) != `{"id":2}` {
			t.Fatalf("calls=%+v", calls)
		}
		return []Reply{{Value: json.RawMessage(`1`)}, {Value: json.RawMessage(`2`)}}, nil
	}))
	if err != nil || string(result.Value) != `[1,2]` {
		t.Fatalf("%s %v", result.Value, err)
	}
}

func TestToolFailureCanBeHandledButHostTerminationCannot(t *testing.T) {
	result, err := Run(t.Context(), `try { await tools.read({}); } catch (e) { return e.output; }`, []string{"read"}, replyScript(func(context.Context, []Call) ([]Reply, error) {
		return []Reply{{Value: json.RawMessage(`{"partial":true}`), Error: "read failed"}}, nil
	}))
	if err != nil || string(result.Value) != `{"partial":true}` {
		t.Fatalf("%s %v", result.Value, err)
	}
	denied := errors.New("denied")
	calls := 0
	_, err = Run(t.Context(), `try { await tools.read({}); } catch {} await tools.read({}); return true;`, []string{"read"}, replyScript(func(context.Context, []Call) ([]Reply, error) {
		calls++
		return nil, denied
	}))
	if !errors.Is(err, denied) || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestInvalidOrUnawaitedProgramsDoNotExecuteTools(t *testing.T) {
	for _, code := range []string{
		`const = ; await tools.read({});`,
		`tools.read({}); return 1;`,
		`await new Promise(() => {});`,
		`await tools.read({value: 1n});`,
	} {
		t.Run(code, func(t *testing.T) {
			_, err := Run(t.Context(), code, []string{"read"}, replyScript(func(context.Context, []Call) ([]Reply, error) { t.Fatal("unexpected effect"); return nil, nil }))
			if err == nil {
				t.Fatal("invalid program succeeded")
			}
		})
	}
}

func TestRuntimeHasNoAmbientHostCapabilities(t *testing.T) {
	result, err := Run(t.Context(), `return [typeof process, typeof require, typeof fetch, typeof std, typeof os];`, nil, nil)
	if err != nil || string(result.Value) != `["undefined","undefined","undefined","undefined","undefined"]` {
		t.Fatalf("%s %v", result.Value, err)
	}
	if _, err := Run(t.Context(), `return await import("std");`, nil, nil); err == nil {
		t.Fatal("host module loader was exposed")
	}
}

func TestRuntimeCancellationAndResourceLimits(t *testing.T) {
	t.Run("cancellation", func(t *testing.T) {
		for _, code := range []string{`while (true) {}`, `while (true) { await Promise.resolve(); }`} {
			ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
			_, err := Run(ctx, code, nil, nil)
			cancel()
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("%v", err)
			}
		}
	})
	for name, code := range map[string]string{
		"memory": `const arrays = []; while(true) arrays.push(new Uint8Array(1024*1024));`,
		"stack":  `function f() { return f(); } return f();`,
		"output": `return "x".repeat(2*1024*1024);`,
	} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			if _, err := Run(ctx, code, nil, nil); err == nil || errors.Is(err, context.DeadlineExceeded) {
				t.Fatal("resource limit was not enforced")
			}
		})
	}
}

func TestThrownMessagesAreBounded(t *testing.T) {
	_, err := Run(t.Context(), `throw new Error("detail".repeat(200000));`, nil, nil)
	if err == nil {
		t.Fatal("throw did not fail execution")
	}
	if len(err.Error()) > 20<<10 {
		t.Fatalf("unbounded thrown message: %d bytes", len(err.Error()))
	}
}

func BenchmarkCodeComposition(b *testing.B) {
	for b.Loop() {
		_, err := Run(b.Context(), `const r = await tools.read({}); return r.value + 1;`, []string{"read"}, replyScript(func(context.Context, []Call) ([]Reply, error) {
			return []Reply{{Value: json.RawMessage(`{"value":41}`)}}, nil
		}))
		if err != nil {
			b.Fatal(err)
		}
	}
}
