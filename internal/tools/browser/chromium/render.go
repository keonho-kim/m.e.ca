// Purpose: Render one approved main document and capture its live HTML with a hard lifetime.
// Pattern: Single-page DevTools controller over a private Chromium process.
// Usage: WebFetch supplies pre-validated document bytes and a bounded resource loader.
// Related: internal/tools/browser/chromium/resources.go, internal/tools/browser/webfetch/document.go.
package chromium

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/processmemory"
)

type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

// LoadResource is the sole network-capable component of the document controller.
type LoadResource func(ctx context.Context, address, method string, maxBytes int) (Response, error)
type Render func(context.Context, string, Response, LoadResource) (string, error)

const maximumHTML = 2 << 20

func RenderHTML(ctx context.Context, executable, base, address string, main Response, load LoadResource) (html string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 22*time.Second)
	defer cancel()
	memory := processmemory.NewPayloadLease()
	defer memory.Close()
	// A frame, its decoded result, and the encoded response can briefly coexist.
	if err := memory.Resize(4 * maxFrame); err != nil {
		return "", err
	}
	browser, closeBrowser, err := start(ctx, executable, base)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, closeBrowser()) }()
	p := page{browser: browser, address: address, main: main, load: load}
	return p.render()
}

type page struct {
	browser                         *pipe
	address, session, frame, loaded string
	main                            Response
	load                            LoadResource
	served                          bool
	requests, bytes                 int
}

func (p *page) render() (string, error) {
	if err := p.browser.call("", "Browser.setDownloadBehavior", map[string]string{"behavior": "deny"}, nil, nil); err != nil {
		return "", err
	}
	target, err := p.documentTarget()
	if err != nil {
		return "", err
	}
	var attached struct{ SessionID string }
	if err := p.browser.call("", "Target.attachToTarget", map[string]any{"targetId": target, "flatten": true}, &attached, nil); err != nil {
		return "", err
	}
	p.session = attached.SessionID
	for _, command := range []struct {
		method string
		params any
	}{
		{"Page.enable", nil},
		{"Page.setLifecycleEventsEnabled", map[string]any{"enabled": true}},
		{"Network.setBypassServiceWorker", map[string]any{"bypass": true}},
		{"Network.setCacheDisabled", map[string]any{"cacheDisabled": true}},
		{"Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": restrictDocumentWindow}},
		{"Fetch.enable", map[string]any{"patterns": []map[string]string{{"urlPattern": "*", "requestStage": "Request"}}}},
	} {
		if err := p.call(command.method, command.params, nil); err != nil {
			return "", err
		}
	}
	var tree struct {
		FrameTree struct{ Frame struct{ ID string } }
	}
	if err := p.call("Page.getFrameTree", nil, &tree); err != nil {
		return "", err
	}
	p.frame = tree.FrameTree.Frame.ID
	var navigation struct{ LoaderID, ErrorText string }
	if err := p.call("Page.navigate", map[string]any{"url": p.address}, &navigation); err != nil {
		return "", err
	}
	if navigation.ErrorText != "" || navigation.LoaderID == "" {
		return "", fault.Errorf(diag.WebFetchNavigation)
	}
	if err := p.waitForDocument(navigation.LoaderID); err != nil {
		return "", err
	}
	if !p.served {
		return "", fault.Errorf(diag.WebFetchNavigation)
	}
	var root struct{ Root struct{ NodeID int } }
	if err := p.call("DOM.getDocument", map[string]any{"depth": 0}, &root); err != nil {
		return "", err
	}
	var content struct{ OuterHTML string }
	if err := p.call("DOM.getOuterHTML", map[string]any{"nodeId": root.Root.NodeID}, &content); err != nil {
		return "", err
	}
	if len(content.OuterHTML) > maximumHTML {
		return "", fault.New(diag.ResourceExhausted, diag.WebFetchBodyLimit)
	}
	return content.OuterHTML, nil
}

func (p *page) documentTarget() (string, error) {
	var targets struct {
		TargetInfos []struct{ TargetID, Type, URL string }
	}
	if err := p.browser.call("", "Target.getTargets", nil, &targets, nil); err != nil {
		return "", err
	}
	for _, target := range targets.TargetInfos {
		if target.Type == "page" && target.URL == "about:blank" {
			return target.TargetID, nil
		}
	}
	var target struct{ TargetID string }
	err := p.browser.call("", "Target.createTarget", map[string]any{"url": "about:blank"}, &target, nil)
	return target.TargetID, err
}

// Disable peer transport and new-window entry points before document scripts run.
const restrictDocumentWindow = `for (const name of ["RTCPeerConnection", "webkitRTCPeerConnection"]) {
  Object.defineProperty(globalThis, name, { value: undefined, writable: false, configurable: false });
}
Object.defineProperty(globalThis, "open", { value: () => null, writable: false, configurable: false });`

func (p *page) call(method string, params, result any) error {
	return p.browser.call(p.session, method, params, result, p.event)
}

func (p *page) event(frame message) error {
	if frame.SessionID != p.session {
		return nil
	}
	switch frame.Method {
	case "Fetch.requestPaused":
		return p.resource(frame.Params)
	case "Page.frameNavigated":
		var event struct {
			Frame struct{ ID, URL, UnreachableURL string }
		}
		if err := json.Unmarshal(frame.Params, &event); err != nil {
			return err
		}
		if p.served && event.Frame.ID == p.frame && (event.Frame.UnreachableURL != "" || !sameURL(event.Frame.URL, p.address)) {
			return fault.Errorf(diag.WebFetchNavigation)
		}
	case "Page.lifecycleEvent":
		var event struct{ FrameID, LoaderID, Name string }
		if err := json.Unmarshal(frame.Params, &event); err != nil {
			return err
		}
		if event.FrameID == p.frame && event.Name == "load" {
			p.loaded = event.LoaderID
		}
	}
	return nil
}

func (p *page) waitForDocument(loader string) error {
	var quiet *time.Timer
	defer func() {
		if quiet != nil {
			quiet.Stop()
		}
	}()
	var quietTime <-chan time.Time
	const renderQuiet = 250 * time.Millisecond
	for {
		if p.loaded == loader && quiet == nil {
			quiet = time.NewTimer(renderQuiet)
			quietTime = quiet.C
		}
		select {
		case <-p.browser.ctx.Done():
			return p.browser.ctx.Err()
		case <-quietTime:
			return nil
		case item, ok := <-p.browser.messages:
			frame, err := p.browser.result(item, ok)
			if err != nil {
				return err
			}
			requests := p.requests
			if err := p.event(frame); err != nil {
				return err
			}
			if quiet != nil && requests != p.requests {
				quiet.Reset(renderQuiet)
			}
		}
	}
}
