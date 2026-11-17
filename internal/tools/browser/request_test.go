// Purpose: Keep browser input bounded, non-shell and incapable of selecting unsafe helper flags.
// Pattern: Pure trust-boundary tests.
// Usage: go test ./internal/tools/browser -run TestRequest.
// Related: internal/tools/browser/request.go.
package browser

import (
	"reflect"
	"testing"
)

func TestRequestBoundaries(t *testing.T) {
	for _, r := range []Request{
		{Action: "open", URL: "file:///etc/passwd"}, {Action: "open", URL: "https://user:secret@example.com"}, {Action: "open", URL: "javascript:alert(1)"},
		{Action: "click", Target: "--cdp"}, {View: "snapshot", Selector: "--config"}, {Action: "press", Value: "--auto-connect"}, {Action: "select", Target: "#select", Value: "--profile"},
		{View: "snapshot", Depth: 13}, {Action: "viewport", Width: 100000, Height: 900}, {ResultID: "result", Action: "click", Target: "#button"},
	} {
		if err := r.Validate(); err == nil {
			t.Fatalf("unsafe request accepted: %+v", r)
		}
	}
	fill := Request{Action: "fill", Target: "#input", Value: "--profile /private"}
	if err := fill.Validate(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fill.args(), []string{"fill", "#input", "--", "--profile /private"}) {
		t.Fatal("fill can inject helper flags", fill.args())
	}
	if External("http://127.0.0.1:3000") || External("http://[::1]/") || !External("https://example.com") {
		t.Fatal("incorrect external interaction boundary")
	}
}
