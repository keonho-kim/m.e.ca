// Purpose: Measure native result admission without model, storage or VM costs.
// Pattern: Fixed JSON input with request-owned result state.
// Usage: go test ./internal/agent/run -run '^$' -bench BenchmarkCodeResultAdmission -benchmem.
// Related: internal/agent/run/code_result.go, internal/agent/run/code_dispatch.go.
package run

import (
	"encoding/json"
	"strings"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/codemode"
)

func BenchmarkCodeResultAdmission(b *testing.B) {
	body, _ := json.Marshal(map[string]string{"text": strings.Repeat("x", 1<<20)})
	s := &execution{runner: Runner{Redact: func(v string) string { return v }}, code: &codeExecution{messageID: "code", replies: make(map[string]codemode.Reply)}}
	call := validatedCall{call: toolapi.Call{ID: "read", ExecutionID: "execution", Name: "execute_command"}}
	b.ReportAllocs()
	for b.Loop() {
		if err := s.acceptCodeResult(call, toolapi.Result{Content: body}, nil); err != nil {
			b.Fatal(err)
		}
		clear(s.code.replies)
	}
}
