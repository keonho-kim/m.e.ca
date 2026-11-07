// Purpose: Bind signed Claude content to the exact earlier system, tools and messages.
// Pattern: Stateless continuation envelope with a conversation-prefix digest.
// Usage: The HTTP response binder seals completed blocks; request replay checks their prefix.
// Related: internal/provider/protocol/claude_request.go, internal/provider/protocol/body.go.
package claude

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type Continuation struct {
	Version int     `json:"version"`
	Prefix  string  `json:"prefix"`
	Blocks  []Block `json:"blocks"`
}

func Bind(blocks []byte, prefix string) ([]byte, error) {
	// Blocks were completed and validated by the decoder. Check the array of
	// objects without decoding their large text/signatures into another model.
	var values []struct{}
	if err := json.Unmarshal(blocks, &values); err != nil || values == nil {
		return nil, fmt.Errorf("invalid completed Claude content")
	}
	return json.Marshal(struct {
		Version int             `json:"version"`
		Prefix  string          `json:"prefix"`
		Blocks  json.RawMessage `json:"blocks"`
	}{1, prefix, blocks})
}

func Restore(raw []byte) (Continuation, error) {
	var value Continuation
	if err := json.Unmarshal(raw, &value); err != nil || value.Version != 1 || len(value.Prefix) != 64 || value.Blocks == nil {
		return value, fmt.Errorf("unsupported Claude continuation; original history was preserved")
	}
	if _, err := hex.DecodeString(value.Prefix); err != nil {
		return value, fmt.Errorf("invalid Claude conversation digest; original history was preserved")
	}
	return value, nil
}
