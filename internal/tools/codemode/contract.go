// Purpose: Define bounded code execution and its host call boundary.
// Pattern: Adapter contracts.
// Usage: The agent runtime supplies Dispatch; the embedded VM exchanges JSON values only.
// Related: internal/tools/codemode/engine.go, internal/agent/run/codemode.go.
package codemode

import (
	"context"
	"encoding/json"
	"time"
)

const (
	Name               = "codemode"
	MaxCodeBytes       = 256 << 10
	MaxArgumentBytes   = 4 << 20
	MaxResultBytes     = 1 << 20
	MaxErrorBytes      = 16 << 10
	MaxToolResultBytes = 8 << 20
	MaxCalls           = 256
	MaxPending         = 32
	HeapBytes          = 64 << 20
	MaxEvaluationTime  = 10 * time.Second
)

type Call struct {
	ID        int
	Name      string
	Arguments json.RawMessage
}

type Reply struct {
	Value json.RawMessage `json:"value"`
	Error string          `json:"error,omitempty"`
}

// Deliver settles one committed call and returns newly requested host calls.
// The dispatcher invokes it only on its serial owner, never from a worker.
type Deliver func(int, Reply) ([]Call, error)

// Dispatch owns every started operation until it has recorded and joined it,
// including when delivery fails or cancellation stops JavaScript execution.
type Dispatch func(context.Context, []Call, Deliver) error

type Result struct {
	Value json.RawMessage
	Calls int
}
