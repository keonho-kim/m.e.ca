// Purpose: Verify runtime limits are tied to exact native instances and all routed deployments.
// Pattern: HTTP contract tests.
// Usage: go test ./internal/provider/httpclient -run TestServingLimits.
// Related: internal/provider/discovery/serving_limits.go, internal/provider/discovery/model_settings.go.
package httpclient

import (
	"context"
	"fmt"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"net/http"
	"testing"
)

func TestServingLimits(t *testing.T) {
	for _, tc := range []struct {
		name, body    string
		input, output int
	}{
		{"all known", `{"data":[{"model_name":"alias","model_info":{"max_input_tokens":1000,"max_output_tokens":200}},{"model_name":"alias","model_info":{"max_input_tokens":500,"max_output_tokens":100}}]}`, 500, 100},
		{"partially unknown", `{"data":[{"model_name":"alias","model_info":{"max_input_tokens":1000,"max_output_tokens":200}},{"model_name":"alias","model_info":{}}]}`, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := mockClient(t, connection.LiteLLM, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, tc.body) })
			info, err := prepareModel(c, context.Background(), "alias")
			if err != nil || info.MaxInput != tc.input || info.MaxOutput != tc.output || info.ContextWindow != 0 {
				t.Fatalf("limits=%+v err=%v", info, err)
			}
		})
	}
}
