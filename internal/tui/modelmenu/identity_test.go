// Purpose: Verify configured and server context limits remain distinct in model information.
// Pattern: Presentation contract test.
// Usage: go test ./internal/tui/modelmenu.
// Related: internal/tui/modelmenu/identity.go.
package modelmenu

import (
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"strings"
	"testing"
)

func TestIdentityContextBudgets(t *testing.T) {
	selected := connection.Selection{Model: "alias", Details: &connection.ModelInfo{ID: "alias", ContextWindow: 8192}}
	text := strings.Join(IdentityLines(selected, 262144, func(s string) string { return s }), "\n")
	for _, want := range []string{"설정 컨텍스트 예산: 262144", "적용 컨텍스트 예산: 8192", "서빙 컨텍스트 한도: 8192"} {
		if !strings.Contains(text, want) {
			t.Fatal(text)
		}
	}
}
