// Purpose: Prepare the optional browser runtime and verify a private round trip through the helper.
// Pattern: Explicit install use case over the same provisioner and browser session used by tools.
// Usage: meca install --ext browser invokes PrepareAndCheck under the application state lease.
// Related: internal/tools/browser/session.go, internal/tools/browser/runtime/install.go, cmd/meca/install.go.
package browser

import (
	"context"
	"errors"
	"time"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/tools/browser/runtime"
)

func PrepareAndCheck(ctx context.Context, state string) (runtime.Paths, error) {
	session := New(state, "", func(text string) string { return text }, nil, nil)
	session.prepare = runtime.Install
	if err := session.start(ctx); err != nil {
		return runtime.Paths{}, err
	}
	paths := runtime.Paths{Agent: session.paths.Agent, Browser: session.paths.Browser}
	if err := checkBlankPage(ctx, session); err != nil {
		if ctx.Err() != nil {
			return runtime.Paths{}, err
		}
		return runtime.Paths{}, errors.Join(fault.New(diag.Unavailable, diag.BrowserProbeFailed, fault.PublicMessage(err, session.redact)), err)
	}
	return paths, nil
}

func checkBlankPage(ctx context.Context, session *Session) (err error) {
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
		defer cancel()
		err = errors.Join(err, session.Close(cleanup))
	}()
	// This fixed internal URL has no web-server, project, or credential dependency.
	opened, err := session.invoke(ctx, "open", "about:blank")
	if err != nil {
		return err
	}
	opened.Close()
	page, err := session.Run(ctx, Request{View: "snapshot"}, "")
	if err != nil {
		return err
	}
	defer page.Close()
	if page.URL != "about:blank" {
		return fault.Errorf(diag.BrowserProbePage)
	}
	return nil
}
