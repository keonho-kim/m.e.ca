// Purpose: Bind browser operations to native approval, credential redaction and vision contracts.
// Pattern: Prepared local tool adapter over a session-owned browser.
// Usage: Executor.Prepare dispatches browser calls here; runner cleanup calls EndBrowser.
// Related: internal/tools/browser/session.go, internal/tools/local/catalog/browser.go.
package local

import (
	"context"
	"encoding/json"

	tool "github.com/keonho-kim/meca/internal/agent/tool"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/media/imagefile"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tools/browser"
)

type browserOperation struct {
	operation
	request browser.Request
	address string
}

func (op *browserOperation) prepareBrowser() error {
	if op.settings.Scope != policy.System {
		return fault.Errorf(diag.BrowserScope)
	}
	in := op.input
	op.request = browser.Request{Action: in.Action, URL: in.URL, Target: in.Target, Value: in.Value, Direction: in.Direction, View: in.View, Selector: in.Selector, ResultID: in.ResultID, Revision: in.Revision, Width: in.Width, Height: in.Height, Pixels: in.Pixels, Depth: in.Depth}
	if in.Offset != nil {
		op.request.Offset = *in.Offset
	}
	if in.Limit != nil {
		op.request.Limit = *in.Limit
	}
	if op.spec.Name == "browser_inspect" && in.Action != "" || op.spec.Name == "browser_action" && (in.View != "" || in.ResultID != "") {
		return fault.Errorf(diag.BrowserInput, diag.Text(diag.BrowserWrongTool))
	}
	if err := op.request.Validate(); err != nil {
		return err
	}
	op.address = op.executor.browser.Current()
	op.approval.Target = op.executor.redact(op.address)
	if in.Action == "open" {
		op.approval.Target = op.executor.redact(in.URL)
	}
	if err := op.browserApproval(); err != nil {
		return err
	}
	preview, err := json.Marshal(struct {
		Action, URL, Target, Value, Direction string
		Pixels, Width, Height                 int
	}{in.Action, op.approval.Target, in.Target, in.Value, in.Direction, in.Pixels, in.Width, in.Height})
	if err != nil {
		return err
	}
	op.approval.Preview = op.executor.redact(string(preview))
	return nil
}

// browserApproval binds interactions to the page observed during preparation.
func (op *browserOperation) browserApproval() error {
	if op.request.Interaction() && op.address == "" {
		return fault.Errorf(diag.BrowserClosed)
	}
	op.approval.Required = op.request.Interaction() && op.settings.Permission != policy.PermissionAllMighty
	if op.request.Interaction() && browser.External(op.address) || op.request.Action == "open" && browser.External(op.request.URL) {
		op.approval.UserRequired = true
		op.approval.Reasons = []string{diag.Text(diag.BrowserTransfer)}
	} else if op.approval.Required {
		op.approval.Reasons = []string{diag.Text(diag.BrowserInteraction)}
	}
	return nil
}

func (op *browserOperation) Execute(ctx context.Context, approved bool) (tool.Result, error) {
	if err := op.claim(ctx, approved); err != nil {
		return tool.Result{}, err
	}
	if op.request.View == "screenshot" && !op.executor.skillImageInput.Load() {
		return tool.Result{}, fault.Errorf(diag.BrowserImage)
	}
	page, err := op.executor.browser.Run(ctx, op.request, op.address)
	if err != nil {
		return tool.Result{}, err
	}
	defer page.Close()
	result := tool.Result{Summary: op.spec.Label}
	if len(page.Image) > 0 {
		image, err := imagefile.Decode(ctx, "browser-viewport.png", page.Image)
		if err != nil {
			return tool.Result{}, err
		}
		result.Images = append(result.Images, image)
	}
	result.Content, err = page.ToolContent()
	return result, err
}
func (e *Executor) EndBrowser(ctx context.Context) error {
	if e.browser == nil {
		return nil
	}
	return e.browser.Close(ctx)
}
