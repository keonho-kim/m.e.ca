// Purpose: Verify configured and server context limits remain distinct in model information.
// Pattern: Presentation contract test.
// Usage: go test ./internal/tui/modelmenu.
// Related: internal/tui/modelmenu/identity.go.
package modelmenu

import (
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/identity"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"strings"
	"testing"
)

func TestIdentityContextBudgets(t *testing.T) {
	selected := connection.Selection{Model: "alias", Details: &connection.ModelInfo{ID: "alias", ContextWindow: 8192}}
	text := strings.Join(IdentityLines(selected, 262144, func(s string) string { return s }, locale.Korean), "\n")
	for _, want := range []string{"설정 컨텍스트 예산: 262144", "적용 컨텍스트 예산: 8192", "서빙 컨텍스트 한도: 8192"} {
		if !strings.Contains(text, want) {
			t.Fatal(text)
		}
	}
}

func TestIdentityControlRegistrationDoesNotClaimDiagnosticSuccess(t *testing.T) {
	selected := connection.Selection{Model: "Ornith-1.5-35B-A3B", ThinkingSupported: true, Identity: &identity.Report{}}
	for _, lang := range []locale.Language{locale.Korean, locale.English} {
		text := strings.Join(IdentityLines(selected, 262144, func(s string) string { return s }, lang), "\n")
		if !strings.Contains(text, lang.Text("Available through server metadata / provider API")) || strings.Contains(text, "연결 진단에서 검증됨") || strings.Contains(text, "Verified by connection diagnostics") {
			t.Fatal(text)
		}
	}
}
