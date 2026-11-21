// Purpose: Define provider-independent model generation and conversation messages.
// Pattern: Consumer-owned model I/O contract.
// Usage: Provider adapters implement Generator; task and persistence code use these messages.
// Related: internal/agent/run/generate.go, internal/provider/protocol/request.go, internal/agent/tool/call.go.

package model

import (
	"context"
	"time"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"

	"github.com/keonho-kim/meca/internal/agent/media"
	"github.com/keonho-kim/meca/internal/agent/team/source"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

const MaxResponseBytes = 256 * 1024 * 1024

type Message struct {
	Presentation string         `json:"presentation,omitempty"`
	Source       *source.Public `json:"source,omitempty"`
	native       *nativeIdentity
	Images       []media.Image `json:"images,omitempty"`
	// ReasoningState is opaque continuation data owned by the source provider and model.
	ReasoningState  []byte `json:"reasoning_state,omitempty"`
	Reasoning       string `json:"reasoning,omitempty"`
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	ID              string `json:"id,omitempty"`
	Role            string `json:"role"`
	Content         string `json:"content"`
	Incomplete      bool   `json:"incomplete,omitempty"`
	// PreserveObservations marks assistant records that do not consume input observations.
	PreserveObservations bool           `json:"preserve_observations,omitempty"`
	Calls                []toolapi.Call `json:"calls,omitempty"`
	ToolCallID           string         `json:"tool_call_id,omitempty"`
	ToolName             string         `json:"tool_name,omitempty"`
	ProviderID           string         `json:"provider_id,omitempty"`
	ModelID              string         `json:"model_id,omitempty"`
	Thinking             *bool          `json:"thinking,omitempty"`

	// TransientObservation supplies detailed evidence for the next completed model decision.
	TransientObservation bool `json:"transient_observation,omitempty"`
}

type Request struct {
	Tools           []toolapi.Definition
	Temperature     *float64
	IdleTimeout     time.Duration
	ReasoningEffort string
	ModelID         string
	Messages        []Message
	MaxOutput       int
	Thinking        *bool

	// OutputContextPercent further bounds output using the effective context window.
	OutputContextPercent uint8
}

type Response struct {
	ReasoningState []byte
	Reasoning      string
	Text           string
	Calls          []toolapi.Call
}

// ToolCallError reports unusable model tool syntax before any operation exists.
type ToolCallError struct{ Detail string }

func (e *ToolCallError) Error() string {
	return diagcatalog.Text(diagcatalog.InvalidToolCall, e.Detail)
}

type Event struct {
	// DecodeDuration is the provider-reported duration for the accompanying output count.
	DecodeDuration time.Duration
	// ReplaceOutput carries both authoritative channels when completion corrects earlier deltas.
	ReplaceOutput bool
	// Progress marks validated non-display tool or continuation bytes, never a keepalive.
	Progress bool
	// InputBytes is transient text-request size from the transport, never token usage.
	InputBytes     int64
	InputBudget    *InputBudget
	ReasoningState []byte
	Retry          *Retry
	Delta          string
	Thinking       string
	Calls          []toolapi.Call
	Done           bool
	Streaming      bool
	Usage          *TokenUsage
}

// HasOutput reports response content or completion delivered by a generation.
func (e Event) HasOutput() bool {
	return e.Delta != "" || e.Thinking != "" || len(e.Calls) > 0 || e.Done
}

// Stream sends one request whose input and model settings have already been bound.
type Stream func(context.Context, func(Event) error) error

// PreparedGeneration measures a request without starting generation. Send
// owns that request and must be used or discarded before preparing another.
type PreparedGeneration struct {
	Budget InputBudget
	Send   Stream
}

// Retry begins a fresh attempt at an incomplete generation. Deltas after it start at offset zero.
type Retry struct {
	Attempt, MaxAttempts int
	Reason               string
}

// Generator invokes emit serially and joins all callbacks before returning.
// An emit error stops the attempt; callbacks never outlive Generate.
type Generator interface {
	Generate(context.Context, Request, func(Event) error) error
}
