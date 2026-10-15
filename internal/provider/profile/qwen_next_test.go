// Purpose: Verify explicit Flash-Next aliases share controls without rewriting request identities.
// Pattern: Catalogue contract tests.
// Usage: go test ./internal/provider/profile -run TestQwenFlashNext.
// Related: internal/provider/profile/qwen.go.
package profile

import (
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"testing"
)

func TestQwenFlashNextAliases(t *testing.T) {
	for _, name := range []string{"Qwen/Qwen3.8-Flash-Next", "Qwen/Qwen3.8-Flash-Next-FP8", "Qwen3.8-Flash-Next 180B", "Qwen-3.8-180B", "Qwen-3.8-180B-FP8"} {
		p, err := Resolve(name, "")
		if err != nil || p.ID != "qwen3.8-flash-next" || p.TotalB != 180 {
			t.Fatalf("%s: %+v %v", name, p, err)
		}
		for _, effort := range []string{"low", "medium", "xhigh"} {
			if err = p.CheckEffort(connection.VLLM, effort); err != nil {
				t.Fatal(err)
			}
		}
	}
	p, err := Resolve("models/bin/Qwen-3.8-180B-custom", "")
	if err != nil || p.ID != Generic {
		t.Fatal("arbitrary name inferred as Flash-Next")
	}
}
