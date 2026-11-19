// Purpose: Fulfill the renderer's permitted read requests through the MECA transport.
// Pattern: Deny-by-default DevTools request interception with aggregate admission.
// Usage: The renderer registers Fetch before navigating the isolated main frame.
// Related: internal/tools/browser/chromium/render.go, internal/tools/browser/webfetch/client.go.
package chromium

import (
	"encoding/json"
	"net/url"
	"strings"
)

const maxResources = 48
const maxResourcesBytes = 4 << 20
const maxResourceBytes = 1 << 20

type pausedRequest struct {
	RequestID, FrameID, ResourceType string
	Request                          struct{ URL, Method string }
}

func (p *page) resource(raw json.RawMessage) error {
	var event pausedRequest
	if err := json.Unmarshal(raw, &event); err != nil {
		return err
	}
	if event.FrameID != p.frame {
		return p.block(event.RequestID)
	}
	if event.ResourceType == "Document" {
		if p.served || event.Request.Method != "GET" || !sameURL(event.Request.URL, p.address) {
			return p.block(event.RequestID)
		}
		p.served = true
		response := p.main
		p.main = Response{}
		return p.fulfill(event.RequestID, response)
	}
	if !p.served || p.requests >= maxResources || p.load == nil {
		return p.block(event.RequestID)
	}
	switch event.ResourceType {
	case "Script", "Stylesheet", "XHR", "Fetch":
	default:
		return p.block(event.RequestID)
	}
	if event.Request.Method != "GET" && event.Request.Method != "HEAD" {
		return p.block(event.RequestID)
	}
	remaining := min(maxResourceBytes, maxResourcesBytes-p.bytes)
	if remaining <= 0 {
		return p.block(event.RequestID)
	}
	p.requests++
	response, err := p.load(p.browser.ctx, event.Request.URL, event.Request.Method, remaining)
	if err != nil || len(response.Body) > remaining {
		return p.block(event.RequestID)
	}
	p.bytes += len(response.Body)
	return p.fulfill(event.RequestID, response)
}

func sameURL(left, right string) bool {
	a, aErr := url.Parse(left)
	b, bErr := url.Parse(right)
	if aErr != nil || bErr != nil || a.Scheme != b.Scheme || !strings.EqualFold(a.Hostname(), b.Hostname()) {
		return false
	}
	port := func(u *url.URL) string {
		if u.Port() != "" {
			return u.Port()
		}
		if u.Scheme == "https" {
			return "443"
		}
		return "80"
	}
	path := func(u *url.URL) string {
		if u.Path == "" {
			return "/"
		}
		return u.EscapedPath()
	}
	return port(a) == port(b) && path(a) == path(b) && a.RawQuery == b.RawQuery
}

func (p *page) block(request string) error {
	_, err := p.browser.send(p.session, "Fetch.failRequest", map[string]any{"requestId": request, "errorReason": "BlockedByClient"})
	return err
}

func (p *page) fulfill(request string, response Response) error {
	type header struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	var headers []header
	for _, name := range []string{
		"Content-Type", "Content-Language", "Content-Security-Policy",
		"Cross-Origin-Resource-Policy", "Cross-Origin-Embedder-Policy", "Cross-Origin-Opener-Policy",
		"Access-Control-Allow-Origin", "Access-Control-Allow-Credentials", "Location",
	} {
		for _, value := range response.Header.Values(name) {
			headers = append(headers, header{Name: name, Value: value})
		}
	}
	_, err := p.browser.send(p.session, "Fetch.fulfillRequest", struct {
		RequestID       string   `json:"requestId"`
		ResponseCode    int      `json:"responseCode"`
		ResponseHeaders []header `json:"responseHeaders,omitempty"`
		Body            []byte   `json:"body,omitempty"`
	}{request, response.Status, headers, response.Body})
	return err
}
