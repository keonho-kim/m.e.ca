// Purpose: Validate each new group of code-generated calls before admitting its effects.
// Pattern: Serial argument and credential boundary.
// Usage: The code dispatcher queues only validated calls and records rejected groups.
// Related: internal/agent/run/code_dispatch.go, internal/agent/run/call_validation.go.
package run

import (
	"crypto/rand"
	"errors"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/redaction"
	"github.com/keonho-kim/meca/internal/tools/codemode"
)

func (d *codeDispatcher) admit(inputs []codemode.Call) error {
	if len(inputs) == 0 {
		return nil
	}
	s := d.execution
	if err := d.check(); err != nil {
		return err
	}
	tasks := make([]*codeTask, len(inputs))
	problems := make([]error, len(inputs))
	invalid := false
	s.usage.recordTools(len(inputs))
	for i, input := range inputs {
		call := toolapi.Call{ID: "inner_" + rand.Text(), Name: input.Name, Arguments: input.Arguments, ParentExecutionID: s.code.call.ExecutionID}
		if err := redaction.CheckProtected(call.Arguments, s.runner.Redact); err != nil {
			s.code.terminal = faults.New(diag.PermissionDenied, diag.LocalToolArgumentsContainAProtectedCredential)
			call.Arguments, err = redaction.RedactJSON(call.Arguments, s.runner.Redact)
			if err != nil {
				return errors.Join(s.code.terminal, err)
			}
			call = s.identifyCall(call)
			return errors.Join(s.code.terminal, s.recordRejected(d.ctx, validatedCall{call: call}, toolapi.Result{}, s.code.terminal, s.emit))
		}
		v, err := s.validateCall(call)
		if err != nil {
			s.code.invalidArguments = true
			invalid = true
			problems[i] = err
			v.call = call
		}
		v.call = s.identifyCall(v.call)
		tasks[i] = &codeTask{id: input.ID, call: v}
	}
	if !invalid {
		d.pending = append(d.pending, tasks...)
		return nil
	}
	for i, task := range tasks {
		var err error
		if problems[i] != nil {
			err = s.recordRejected(d.ctx, task.call, toolapi.Result{}, problems[i], s.emit)
		} else {
			err = s.skipCalls(d.ctx, []validatedCall{task.call}, skipAfterFailure, "Another call in this wave has invalid arguments; no operation in this wave executed.")
		}
		if err != nil {
			return err
		}
	}
	d.feedback = append(d.feedback, tasks...)
	return nil
}
