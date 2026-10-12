// Purpose: Own validated provider connections and model discovery result values.
// Pattern: Simple Module (model-selection boundary).
// Usage: Used by provider construction, model menus and persisted connections.
// Related: internal/provider/httpclient/client.go, internal/storage/sqlite/selection.go, SYSTEM.md.

package connection

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"strings"
)

type ProviderKind string

const (
	Ollama           ProviderKind = "ollama"
	VLLM             ProviderKind = "vllm"
	SGLang           ProviderKind = "sglang"
	LMStudio         ProviderKind = "lm-studio"
	OpenAICompatible ProviderKind = "openai-compatible"
)

type Connection struct {
	ID            string       `json:"id"`
	Kind          ProviderKind `json:"kind"`
	Endpoint      string       `json:"endpoint"`
	CredentialRef string       `json:"credential_ref"`
}

type Capabilities struct {
	Chat         bool `json:"chat"`
	ToolDialogue bool `json:"tool_dialogue"`
	Streaming    bool `json:"streaming"`
}

type ModelInfo struct {
	ID            string       `json:"id"`
	ContextWindow int          `json:"context_window"`
	MaxOutput     int          `json:"max_output"`
	Capabilities  Capabilities `json:"capabilities"`
}

func New(kind ProviderKind, endpoint string) (Connection, error) {
	var defaultEndpoint string
	switch kind {
	case Ollama:
		defaultEndpoint = "http://localhost:11434"
	case VLLM, SGLang, OpenAICompatible:
		defaultEndpoint = "http://localhost:8000/v1"
	case LMStudio:
		defaultEndpoint = "http://localhost:1234"
	default:
		return Connection{}, fmt.Errorf("unknown provider: %s", kind)
	}
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	if len(endpoint) > 4096 {
		return Connection{}, fmt.Errorf("Endpoint is too long")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Hostname() == "" || parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return Connection{}, fmt.Errorf("Endpoint must be an HTTP(S) URL without credentials, query or fragment")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	endpoint = parsed.String()
	id := sha256.Sum256([]byte(string(kind) + "\n" + endpoint))
	return Connection{ID: fmt.Sprintf("provider_%x", id[:12]), Kind: kind, Endpoint: endpoint}, nil
}
