// Purpose: Expose immutable tool functions while keeping host bindings private.
// Pattern: JavaScript-to-JSON capability bridge.
// Usage: The serial driver drains pending promises outside JavaScript evaluation.
// Related: internal/tools/codemode/engine.go, internal/tools/codemode/contract.go.
package codemode

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"modernc.org/quickjs"
)

type runtime struct {
	ctx       context.Context
	vm        *quickjs.VM
	tools     *quickjs.Value
	deliver   *quickjs.Value
	serialize *quickjs.Value
	pending   []Call
	active    map[int]bool
	calls     int
	names     map[string]bool
	remaining time.Duration
}

func (r *runtime) close() {
	for _, value := range []*quickjs.Value{r.serialize, r.deliver, r.tools} {
		if value != nil {
			value.Free()
		}
	}
}

func (r *runtime) install(names []string) error {
	r.names = make(map[string]bool, len(names))
	r.active = make(map[int]bool)
	for _, name := range names {
		r.names[name] = true
	}
	if err := r.vm.RegisterHostFunc("__mecaCall", r.enqueue); err != nil {
		return runtimeError(err)
	}
	encoded, err := json.Marshal(names)
	if err != nil {
		return runtimeError(err)
	}
	if names == nil {
		encoded = []byte(`[]`)
	}
	return r.step(func() error {
		var err error
		controller, err := r.vm.EvalValue(`(() => {
			const call = globalThis.__mecaCall;
			delete globalThis.__mecaCall;
			const stringify = JSON.stringify, parse = JSON.parse;
			const encodeUTF8 = encodeURIComponent, decodeBytes = unescape;
			const ascii = Function.prototype.call.bind(RegExp.prototype.exec, /^[\x00-\x7f]*$/);
			const ToolError = Error;
			const ToolPromise = Promise;
			const pending = new Map();
			const tools = Object.create(null);
			for (const name of `+string(encoded)+`) {
				tools[name] = (args = {}) => {
					const id = call(name, stringify(args));
					return new ToolPromise((resolve, reject) => pending.set(id, {resolve, reject}));
				};
			}
			return { tools: Object.freeze(tools), serialize(value) {
				const text = stringify(value);
				if (typeof text !== "string") throw new ToolError("unsupported returned JSON value");
				const limit = `+strconv.Itoa(MaxResultBytes)+`;
				if (text.length > limit) throw new ToolError("returned JSON size exceeds " + limit + " UTF-8 bytes");
				if (text.length > limit / 3 && !ascii(text)) {
					// Native codecs count UTF-8 bytes inside the bounded JS heap.
					// JSON.stringify escapes lone surrogates before this conversion.
					if (decodeBytes(encodeUTF8(text)).length > limit) throw new ToolError("returned JSON size exceeds " + limit + " UTF-8 bytes");
				}
				return text;
			}, deliver(id, encoded, error) {
				const entry = pending.get(id);
				pending.delete(id);
				const result = parse(encoded);
				if (error) {
					const failure = new ToolError(error);
					failure.output = result;
					entry.reject(failure);
				} else entry.resolve(result);
			}};
		})()`, quickjs.EvalGlobal)
		if err != nil {
			return err
		}
		defer controller.Free()
		for _, binding := range []struct {
			name   string
			target **quickjs.Value
		}{{"tools", &r.tools}, {"deliver", &r.deliver}, {"serialize", &r.serialize}} {
			atom, err := r.vm.NewAtom(binding.name)
			if err != nil {
				return err
			}
			value, err := controller.GetPropertyValue(atom)
			if err != nil {
				return err
			}
			*binding.target = &value
		}
		return nil
	})
}

func (r *runtime) enqueue(args []any) (any, error) {
	if err := r.ctx.Err(); err != nil {
		return nil, err
	}
	if len(args) != 2 {
		return nil, faults.New(diag.InvalidArgument, diag.CodeInvalidBridge)
	}
	name, nameOK := args[0].(string)
	body, bodyOK := args[1].(string)
	if !nameOK || !bodyOK || len(body) == 0 || body[0] != '{' || len(body) > MaxArgumentBytes || !json.Valid([]byte(body)) {
		return nil, faults.New(diag.InvalidArgument, diag.CodeInvalidBridge)
	}
	if !r.names[name] {
		return nil, faults.New(diag.InvalidArgument, diag.CodeUnknownTool, name)
	}
	if r.calls >= MaxCalls || len(r.active) >= MaxPending {
		return nil, faults.New(diag.ResourceExhausted, diag.CodeResourceLimit, "tool calls")
	}
	r.calls++
	r.active[r.calls] = true
	r.pending = append(r.pending, Call{ID: r.calls, Name: name, Arguments: json.RawMessage(body)})
	return r.calls, nil
}
