// Purpose: Own read-only document rendering within the browser tool session.
// Pattern: Browser lifecycle adapter over the read-only document transport.
// Usage: The local executor prepares and runs web_fetch through this Session.
// Related: internal/tools/browser/session.go, internal/tools/browser/chromium/render.go.
package browser

import (
	"context"
	"errors"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/tools/browser/chromium"
	"github.com/keonho-kim/meca/internal/tools/browser/runtime"
	"github.com/keonho-kim/meca/internal/tools/browser/webfetch"
)

func (s *Session) PrepareDocument(input webfetch.Input) (webfetch.Request, error) {
	request, err := s.documents.Prepare(input)
	if err != nil {
		return webfetch.Request{}, err
	}
	if request.StartsDocument() && !runtime.ChromeAvailable(s.state) {
		return webfetch.Request{}, fault.New(diag.FailedPrecondition, diag.WebFetchBrowser)
	}
	return request, nil
}

func (s *Session) ReadDocument(ctx context.Context, request webfetch.Request) (webfetch.Page, error) {
	if s.documents == nil {
		return webfetch.Page{}, fault.New(diag.FailedPrecondition, diag.WebFetchDisabled)
	}
	select {
	case s.gate <- struct{}{}:
		defer func() { <-s.gate }()
	case <-ctx.Done():
		return webfetch.Page{}, ctx.Err()
	}
	if request.StartsDocument() && !runtime.ChromeAvailable(s.state) {
		return webfetch.Page{}, fault.New(diag.FailedPrecondition, diag.WebFetchBrowser)
	}
	return s.documents.Read(ctx, request, s.renderDocument)
}

func (s *Session) renderDocument(ctx context.Context, address string, main chromium.Response, load chromium.LoadResource) (_ string, err error) {
	release, err := acquireBrowser()
	if err != nil {
		return "", err
	}
	defer release()
	paths, err := runtime.OpenChrome(ctx, s.state)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, paths.Close()) }()
	return chromium.RenderHTML(ctx, paths.Browser, s.base, address, main, load)
}
