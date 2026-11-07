// Purpose: Expose observed tool outcomes without serializing execution-private target or error objects.
// Pattern: Explicit public operation DTO mapping.
// Usage: Complete event detail includes this projection for execution facts and file changes.
// Related: internal/serverapi/detail/event.go, internal/agent/tool/operation.go.
package detail

import "github.com/keonho-kim/meca/internal/agent/tool"

type operation struct {
	Revision int               `json:"revision"`
	Target   string            `json:"target,omitempty"`
	Versions map[string]string `json:"versions,omitempty"`
	Started  bool              `json:"started"`
	RunID    string            `json:"runId,omitempty"`
	CallID   string            `json:"callId,omitempty"`
	Name     string            `json:"name"`
	Status   string            `json:"status"`
	Command  string            `json:"command,omitempty"`
	Result   string            `json:"result,omitempty"`
	Failure  *failure          `json:"failure,omitempty"`
	Process  *process          `json:"process,omitempty"`
	Changes  []change          `json:"changes,omitempty"`
}
type failure struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Next   string `json:"next"`
}
type process struct {
	CommandID  string `json:"commandId"`
	Command    string `json:"command"`
	Diagnostic string `json:"diagnostic,omitempty"`
	Truncated  bool   `json:"truncated"`
	Status     string `json:"status"`
	ExitCode   int    `json:"exitCode"`
}
type change struct {
	Action       string       `json:"action"`
	Path         string       `json:"path"`
	Destination  string       `json:"destination,omitempty"`
	BeforeDigest string       `json:"beforeDigest,omitempty"`
	AfterDigest  string       `json:"afterDigest,omitempty"`
	Lines        []lineChange `json:"lines,omitempty"`
}
type lineChange struct {
	OldStartLine int  `json:"oldStartLine"`
	OldEndLine   int  `json:"oldEndLine"`
	NewStartLine int  `json:"newStartLine"`
	NewEndLine   int  `json:"newEndLine"`
	Removed      bool `json:"removed"`
}

func publicOperation(value tool.Operation) *operation {
	o := &operation{Revision: value.Revision, Target: value.Target, Versions: value.Versions, Started: value.Started, RunID: value.RunID, CallID: value.CallID, Name: value.Name, Status: value.Status, Command: value.Command, Result: value.Result}
	if f := value.Failure; f != nil {
		o.Failure = &failure{f.Code, f.Detail, f.Next}
	}
	if p := value.Process; p != nil {
		o.Process = &process{p.CommandID, p.Command, p.Diagnostic, p.Truncated, p.Status, p.ExitCode}
	}
	for _, c := range value.Changes {
		x := change{Action: c.Action, Path: c.Path, Destination: c.Destination, BeforeDigest: c.BeforeDigest, AfterDigest: c.AfterDigest}
		for _, l := range c.Lines {
			x.Lines = append(x.Lines, lineChange{l.OldStartLine, l.OldEndLine, l.NewStartLine, l.NewEndLine, l.Removed})
		}
		o.Changes = append(o.Changes, x)
	}
	return o
}
