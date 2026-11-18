// Purpose: Reduce a successful browser page to a read receipt for decision and summary contexts.
// Pattern: Pure projection of the tool result contract.
// Usage: The agent acknowledges completed delivery; archived compaction excludes the consumed text.
// Related: internal/agent/run/browser_pages.go, internal/storage/history/compaction.go.
package tool

import "encoding/json"

// A result ID is produced by text observations. Failures and screenshots use other envelopes.
type browserReceipt struct {
	Read       bool   `json:"read"`
	ResultID   string `json:"result_id,omitempty"`
	Revision   string `json:"revision,omitempty"`
	NextOffset int    `json:"next_offset,omitempty"`
}

// ReadBrowserPage returns only the cursor required to continue a previously delivered result.
func ReadBrowserPage(content string) (string, bool) {
	var receipt browserReceipt
	if err := json.Unmarshal([]byte(content), &receipt); err != nil || receipt.Read || receipt.ResultID == "" {
		return "", false
	}
	if receipt.NextOffset == 0 {
		return `{"read":true}`, true
	}
	receipt.Read = true
	encoded, _ := json.Marshal(receipt)
	return string(encoded), true
}
