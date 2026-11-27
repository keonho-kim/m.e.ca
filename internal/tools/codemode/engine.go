// Purpose: Execute isolated JavaScript and resume it after each committed host result.
// Pattern: Serial interpreter driver with an explicit host I/O boundary.
// Usage: Agent execution calls Run and owns every effect through Dispatch.
// Related: internal/tools/codemode/runtime.go, internal/tools/codemode/transform.go.
package codemode

import (
	"context"
	"encoding/json"
	"errors"
	"time"
	"unicode/utf8"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/processmemory"
	"modernc.org/quickjs"
)

// Run executes an async function body. Only Dispatch can perform host effects.
func Run(ctx context.Context, code string, names []string, dispatch Dispatch) (result Result, err error) {
	if err := ctx.Err(); err != nil {
		return result, err
	}
	release, err := processmemory.AcquireCode(ctx, HeapBytes)
	if err != nil {
		return result, err
	}
	defer release()
	compiled, err := transform(code)
	if err != nil {
		return result, err
	}
	vm, err := quickjs.NewVM()
	if err != nil {
		return result, runtimeError(err)
	}
	defer func() {
		if closeErr := vm.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()
	vm.SetMemoryLimit(HeapBytes)
	vm.SetMaxStackSize(256)
	vm.SetCanBlock(false)
	stopped := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { vm.Interrupt(); close(stopped) })
	defer func() {
		if !stop() {
			<-stopped
		}
	}()
	r := &runtime{ctx: ctx, vm: vm, remaining: MaxEvaluationTime}
	defer r.close()
	if err := r.install(names); err != nil {
		return result, err
	}
	var promise quickjs.Value
	err = r.step(func() error {
		fn, err := vm.EvalValue(compiled, quickjs.EvalGlobal)
		if err != nil {
			return err
		}
		defer fn.Free()
		promise, err = fn.CallValue(quickjs.UndefinedValue, *r.tools)
		return err
	})
	if err != nil {
		return result, err
	}
	defer promise.Free()
	var settled bool
	var rejected error
	observer, err := promise.Then(func(value quickjs.Value) {
		settled = true
		result.Defined = !value.IsUndefined()
		if !result.Defined {
			result.Value = json.RawMessage(`null`)
			return
		}
		encoded, err := r.serialize.Call(quickjs.UndefinedValue, value)
		if err != nil {
			rejected = err
			return
		}
		text, ok := encoded.(string)
		if !ok {
			rejected = faults.New(diag.InvalidArgument, diag.CodeInvalidResult, "unsupported value")
			return
		}
		result.Value = json.RawMessage(text)
		if rejected == nil && !json.Valid(result.Value) {
			rejected = faults.New(diag.InvalidArgument, diag.CodeInvalidResult, "unsupported value")
		}
		if len(result.Value) > MaxResultBytes {
			result.Value = nil
			rejected = faults.New(diag.ResourceExhausted, diag.CodeResourceLimit, "returned JSON size")
		}
	}, func(value quickjs.Value) {
		settled = true
		rejected = quickjs.ErrorFromValue(value)
	})
	if err != nil {
		return result, runtimeError(err)
	}
	defer observer.Free()
	advance := func() ([]Call, error) {
		if err := r.step(func() error { _, err := vm.ExecutePendingJobs(); return err }); err != nil {
			return nil, err
		}
		if settled {
			if len(r.active) != 0 && rejected == nil {
				return nil, faults.New(diag.FailedPrecondition, diag.CodeUnawaited)
			}
			return nil, runtimeError(rejected)
		}
		if len(r.active) == 0 {
			return nil, faults.New(diag.FailedPrecondition, diag.CodeUnresolved)
		}
		pending := r.pending
		r.pending = nil
		result.Calls += len(pending)
		return pending, nil
	}
	calls, err := advance()
	if err != nil || settled {
		return result, err
	}
	if dispatch == nil {
		return result, faults.New(diag.FailedPrecondition, diag.CodeInvalidBridge)
	}
	err = dispatch(ctx, calls, func(id int, reply Reply) ([]Call, error) {
		if !r.active[id] {
			return nil, faults.New(diag.Internal, diag.CodeDispatchMismatch)
		}
		delete(r.active, id)
		if len(reply.Value) == 0 {
			reply.Value = json.RawMessage(`null`)
		}
		if len(reply.Value) > MaxToolResultBytes {
			return nil, faults.New(diag.ResourceExhausted, diag.CodeResourceLimit, "tool result size")
		}
		if err := r.step(func() error {
			_, err := r.deliver.Call(quickjs.UndefinedValue, id, string(reply.Value), reply.Error)
			return err
		}); err != nil {
			return nil, err
		}
		return advance()
	})
	if err == nil && (!settled || len(r.active) != 0) {
		err = faults.New(diag.Internal, diag.CodeDispatchMismatch)
	}
	return result, err
}

func (r *runtime) step(fn func() error) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	if r.remaining <= 0 {
		return faults.New(diag.ResourceExhausted, diag.CodeResourceLimit, "JavaScript execution time")
	}
	if err := r.vm.SetEvalTimeout(r.remaining); err != nil {
		return runtimeError(err)
	}
	started := time.Now()
	err := fn()
	r.remaining -= time.Since(started)
	if r.ctx.Err() != nil {
		return r.ctx.Err()
	}
	return runtimeError(err)
}

func runtimeError(err error) error {
	if err == nil {
		return nil
	}
	detail := err.Error()
	if len(detail) > MaxErrorBytes {
		end := MaxErrorBytes
		for end > 0 && !utf8.RuneStart(detail[end]) {
			end--
		}
		detail = detail[:end] + " [message shortened]"
	}
	return faults.New(diag.ExecutionFailed, diag.CodeRuntime, detail)
}
