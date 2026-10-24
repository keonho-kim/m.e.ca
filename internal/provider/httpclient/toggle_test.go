// Purpose: Reject servers that ignore OFF on reasoning tasks despite answering the ordinary chat probe.
// Pattern: Capability diagnostic contract test.
// Usage: go test ./internal/provider/httpclient -run TestReasoningToggleChecksOff.
// Related: internal/provider/diagnostic/check.go, internal/provider/diagnostic/effort.go.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/protocol"
	"net/http"
	"strings"
	"testing"
)

func TestReasoningToggleChecksOff(t *testing.T) {
	c := mockClient(t, connection.Ollama, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/show" {
			fmt.Fprint(w, `{"thinking":{"values":[false,true]}}`)
			return
		}
		if r.URL.Path == "/api/ps" {
			fmt.Fprint(w, `{"models":[]}`)
			return
		}
		var body struct{ Messages []protocol.Message }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		answer, reason := "Hello!", ""
		if strings.Contains(body.Messages[len(body.Messages)-1].Content, "17 times 23") {
			answer, reason = "391", "always thinking"
		}
		if err := json.NewEncoder(w).Encode(map[string]any{"message": map[string]string{"content": answer, "thinking": reason}, "done": true}); err != nil {
			t.Error(err)
		}
	})
	_, err := diagnose(c, context.Background(), "alias")
	if err == nil || !strings.Contains(err.Error(), "OFF") {
		t.Fatalf("ignored OFF was accepted: %v", err)
	}
}
