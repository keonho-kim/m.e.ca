// Purpose: Verify PFC literal fields and generated content select different role policies.
// Pattern: Role selection contract test.
// Usage: go test ./internal/agent/inference.
// Related: internal/agent/inference/role.go, internal/agent/inference/settings.go.
package inference

import (
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"testing"
)

func TestFieldRoles(t *testing.T) {
	for _, test := range []struct {
		name string
		kind toolapi.ParameterKind
		want Role
	}{
		{"path", toolapi.LiteralPath, PFCValue}, {"start_anchor", toolapi.LineAnchor, PFCValue}, {"limit", toolapi.Integer, PFCValue}, {"pattern", toolapi.Text, PFCValue}, {"content", toolapi.Text, PFCContent}, {"new_text", toolapi.Text, PFCContent}, {"command", toolapi.Text, PFCContent},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := FieldRole(toolapi.Parameter{Name: test.name, Kind: test.kind}); got != test.want {
				t.Fatalf("role=%s want=%s", got, test.want)
			}
		})
	}
}
