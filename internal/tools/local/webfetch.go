// Purpose: Bind a WebFetch read to one native tool invocation.
// Pattern: Prepared read-only tool operation.
// Usage: Executor.prepare compiles the requested URL or cursor; the runtime executes it once.
// Related: internal/tools/local/catalog/browser.go, internal/tools/browser/webfetch/documents.go.
package local

import (
	"context"

	json "github.com/goccy/go-json"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/browser/webfetch"
)

type webFetchOperation struct {
	operation
	request webfetch.Request
}

func (op *webFetchOperation) prepare() error {
	input := op.input
	request, err := op.executor.browser.PrepareDocument(webfetch.Input{URL: input.URL, ResultID: input.ResultID, Revision: input.Revision, Offset: input.Offset, Limit: input.Limit})
	if err != nil {
		return err
	}
	op.request = request
	return nil
}

func (op *webFetchOperation) Execute(ctx context.Context, approved bool) (toolapi.Result, error) {
	if err := op.claim(ctx, approved); err != nil {
		return toolapi.Result{}, err
	}
	page, err := op.executor.browser.ReadDocument(ctx, op.request)
	if err != nil {
		return toolapi.Result{}, err
	}
	content, err := json.MarshalWithOption(page, json.DisableHTMLEscape())
	return toolapi.Result{Content: content, Summary: op.spec.Label}, err
}
