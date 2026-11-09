// Purpose: Bind signed Claude content to the exact earlier system, tools and messages.
// Pattern: Stateless continuation envelope with a conversation-prefix digest.
// Usage: The HTTP response binder seals completed blocks; request replay checks their prefix.
// Related: internal/provider/protocol/claude_request.go, internal/provider/protocol/body.go.
package claude

import (
	"encoding/hex"
	"encoding/json"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
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
		return nil, faults.Errorf(diagcatalog.ClaudeInvalidCompletedClaudeContent)
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
		return value, faults.Errorf(diagcatalog.ClaudeUnsupportedClaudeContinuationOriginalHistoryWasPreserved)
	}
	if _, err := hex.DecodeString(value.Prefix); err != nil {
		return value, faults.Errorf(diagcatalog.ClaudeInvalidClaudeConversationDigestOriginalHistoryWas)
	}
	return value, nil
}
