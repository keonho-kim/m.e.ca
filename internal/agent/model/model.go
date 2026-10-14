// Purpose: Define provider-independent model generation and conversation messages.
// Pattern: Consumer-owned model I/O contract.
// Usage: Provider adapters implement Generator; task and persistence code use these messages.
// Related: internal/agent/run/generate.go, internal/provider/protocol/request.go, internal/agent/tool/call.go.

package model

import (
	"context"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"time"
)

const MaxResponseBytes = 4 * 1024 * 1024

type Message struct {
	ReasoningEffort string         `json:"reasoning_effort,omitempty"`
	ID              string         `json:"id,omitempty"`
	Role            string         `json:"role"`
	Content         string         `json:"content"`
	Calls           []toolapi.Call `json:"calls,omitempty"`
	ToolCallID      string         `json:"tool_call_id,omitempty"`
	ToolName        string         `json:"tool_name,omitempty"`
	ProviderID      string         `json:"provider_id,omitempty"`
	ModelID         string         `json:"model_id,omitempty"`
	Thinking        *bool          `json:"thinking,omitempty"`
	Planning        bool           `json:"planning,omitempty"`
}

type Request struct {
	Temperature     *float64
	IdleTimeout     time.Duration
	ReasoningEffort string
	ModelID         string
	Messages        []Message
	MaxOutput       int
	Thinking        *bool
}

type Event struct {
	Retry     *Retry
	Delta     string
	Thinking  string
	Calls     []toolapi.Call
	Done      bool
	Streaming bool
	Usage     *TokenUsage
}

// Retry begins a fresh generation attempt; consumers discard its previous partial answer.
type Retry struct {
	Attempt, MaxAttempts int
	Reason               string
}

type Generator interface {
	Generate(context.Context, Request, func(Event) error) error
}
