// Purpose: Verify each checkpoint exposes only its documented Thinking effort values.
// Pattern: Model-profile policy tests.
// Usage: go test ./internal/provider/profile -run TestEffort.
// Related: internal/provider/profile/effort.go, internal/provider/profile/qwen.go, internal/provider/profile/nemotron.go.
package profile

import (
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"reflect"
	"testing"
)

func TestEffortValuesAreCheckpointSpecific(t *testing.T) {
	for _, test := range []struct {
		model    string
		provider modelconn.ProviderKind
		want     []string
	}{
		{"Qwen/Qwen3.8-27B", modelconn.VLLM, []string{"low", "medium", "xhigh"}},
		{"Qwen/Qwen3.8-27B", modelconn.SGLang, []string{"low", "medium", "xhigh"}},
		{"Qwen/Qwen3.6-27B", modelconn.VLLM, nil},
		{"ornith-ai/Ornith-1.5-9B", modelconn.VLLM, nil},
		{"nvidia/NVIDIA-Nemotron-3-Super-120B-A12B-BF16", modelconn.VLLM, []string{"low", "full"}},
		{"nvidia/NVIDIA-Nemotron-3.5-Lightning-30B-A3B-BF16", modelconn.VLLM, nil},
		{"deepseek-ai/DeepSeek-V4-Flash", modelconn.VLLM, []string{"high", "max"}},
		{"deepseek-ai/DeepSeek-V4-Flash", modelconn.SGLang, nil},
		{"google/gemma-4-31B-it", modelconn.VLLM, nil},
		{"Qwen/Qwen3.8-27B", modelconn.Ollama, nil},
		{"Qwen/Qwen3.8-27B", modelconn.LiteLLM, nil},
	} {
		t.Run(test.model+"/"+string(test.provider), func(t *testing.T) {
			p, err := Resolve(test.model, "")
			if err != nil {
				t.Fatal(err)
			}
			if got := p.EffortValues(test.provider); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("levels=%v; want %v", got, test.want)
			}
			if err := p.CheckEffort(test.provider, ""); err != nil {
				t.Fatal("default should omit explicit effort", err)
			}
			if err := p.CheckEffort(test.provider, "invented"); err == nil {
				t.Fatal("unknown effort accepted")
			}
		})
	}
}
