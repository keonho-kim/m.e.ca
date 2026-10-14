// Purpose: Own supported provider names and their default endpoints.
// Pattern: Fixed provider catalog.
// Usage: Model menus and connection normalization use the same declared providers.
// Related: internal/provider/connection/connection.go, internal/tui/session/menus.go.

package connection

type ProviderKind string

const (
	Ollama           ProviderKind = "ollama"
	VLLM             ProviderKind = "vllm"
	SGLang           ProviderKind = "sglang"
	LMStudio         ProviderKind = "lm-studio"
	LiteLLM          ProviderKind = "litellm"
	OpenAICompatible ProviderKind = "openai-compatible"
)

type Provider struct {
	Label    string
	Kind     ProviderKind
	Endpoint string
}

func Providers() []Provider {
	return []Provider{
		{"Ollama", Ollama, "http://localhost:11434"},
		{"vLLM", VLLM, "http://localhost:8000/v1"},
		{"SGLang", SGLang, "http://localhost:8000/v1"},
		{"LM Studio", LMStudio, "http://localhost:1234"},
		{"LiteLLM", LiteLLM, "http://localhost:4000/v1"},
		{"OpenAI Compatible", OpenAICompatible, "http://localhost:8000/v1"},
	}
}
