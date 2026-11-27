// Purpose: Build deterministic code-mode model calls for cross-layer integration fixtures.
// Pattern: Test fixture builder, imported only by tests.
// Usage: Runtime, transport, terminal and team tests describe the host operations their scripted model requests.
// Related: internal/agent/tool/call.go, internal/agent/run/codemode_test.go.
package modeltest

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/keonho-kim/meca/internal/agent/tool"
)

func Calls(calls []tool.Call) []tool.Call {
	if len(calls) == 0 {
		return nil
	}
	parts := make([]string, len(calls))
	for i, call := range calls {
		parts[i] = fmt.Sprintf("tools[%q](%s)", call.Name, call.Arguments)
	}
	code := "return await " + parts[0] + ";"
	if len(parts) > 1 {
		code = "return await Promise.all([" + strings.Join(parts, ",") + "]);"
	}
	body, _ := json.Marshal(map[string]string{"code": code})
	return []tool.Call{{ID: calls[0].ID, Name: "codemode", Arguments: body}}
}
