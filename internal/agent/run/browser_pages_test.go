// Purpose: Verify a model sees each observed page once across decisions and restored history.
// Pattern: Scripted agent with browser page fixtures and a durable journal.
// Usage: go test ./internal/agent/run -run TestBrowserPage.
// Related: internal/agent/run/browser_pages.go, internal/agent/run/context_memory.go.
package run

import (
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

type pageTools struct{ *executor }

func (e pageTools) Definitions() []toolapi.Definition {
	return append(e.executor.Definitions(), toolapi.Definition{Name: "browser_inspect", Effect: policy.Command, Description: "Read a page", Parameters: []toolapi.Parameter{{Name: "offset", Kind: toolapi.Integer, Max: 16384}}})
}
func containsPage(messages []modelapi.Message, text string) bool {
	for _, m := range messages {
		if m.Role == "tool" && strings.Contains(m.Content, text) {
			return true
		}
	}
	return false
}
func TestBrowserPageDeliveredOnce(t *testing.T) {
	page1 := `{"result_id":"result_A","revision":"r","next_offset":2048,"text":"first body"}`
	page2 := `{"result_id":"result_A","revision":"r","offset":2048,"text":"second body"}`
	g := &generator{replies: []string{nativeCall("browser_inspect", map[string]any{}), nativeCall("browser_inspect", map[string]any{"offset": 2048}), "done"}}
	e := &executor{outcomes: []toolapi.Result{{Content: []byte(page1)}, {Content: []byte(page2)}}}
	j := &journal{}
	r := fixture(g, e, j)
	r.Tools = pageTools{e}
	r.RedactionVersion = func() int { return 0 }
	if _, err := r.Run(t.Context(), "Inspect", approve, emit); err != nil {
		t.Fatal(err)
	}
	if len(g.requests) != 3 || !containsPage(g.requests[1].Messages, "first body") || containsPage(g.requests[2].Messages, "first body") || !containsPage(g.requests[2].Messages, "second body") {
		t.Fatal("page acknowledgement broke delivery")
	}
	if !containsPage(j.messages, "first body") || !containsPage(j.messages, "second body") {
		t.Fatal("original evidence lost")
	}
	g.replies = append(g.replies, "done again")
	if _, err := r.Run(t.Context(), "Continue", approve, emit); err != nil {
		t.Fatal(err)
	}
	restored := g.requests[len(g.requests)-1]
	if containsPage(restored.Messages, "first body") || containsPage(restored.Messages, "second body") {
		t.Fatal("consumed page rehydrated into model request")
	}
}

func TestBrowserPageUnacknowledgedOnIncompleteReply(t *testing.T) {
	s := &execution{messages: []modelapi.Message{
		{Role: "tool", ToolName: toolapi.BrowserInspect, Content: `{"result_id":"id","text":"pending page"}`, TransientObservation: true},
		{Role: "assistant", Incomplete: true, Content: "partial"},
	}}
	s.restoreBrowserReads()
	if !containsPage(s.messages, "pending page") {
		t.Fatal("interrupted generation consumed unread page")
	}
}

func TestBrowserScreenshotDeliveredOnce(t *testing.T) {
	images := []media.Image{{Data: []byte("pixels")}}
	s := &execution{messages: []modelapi.Message{
		{Role: "user", Images: images},
		{Role: "tool", ToolName: "view_image", Images: images},
		{Role: "tool", ToolName: toolapi.BrowserInspect, Content: "screenshot", Images: images, TransientObservation: true},
	}}
	s.acknowledgeBrowserObservations(len(s.messages))
	if m := s.messages[2]; len(m.Images) != 0 || m.TransientObservation || m.Content != toolapi.BrowserObservationRead {
		t.Fatal("delivered browser screenshot still occupies model context")
	}
	if len(s.messages[0].Images) != 1 || len(s.messages[1].Images) != 1 {
		t.Fatal("browser acknowledgement affected other vision inputs")
	}
}
