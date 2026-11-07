// Purpose: Preserve native Claude block metadata and required empty signed fields.
// Pattern: Typed block codec with lossless optional-field retention.
// Usage: Streaming drafts and continuation envelopes share Block's JSON methods.
// Related: internal/provider/protocol/claude/types.go, internal/provider/protocol/claude/continuation.go.
package claude

import "encoding/json"

func (b *Block) UnmarshalJSON(body []byte) error {
	type plain Block
	var value plain
	if err := json.Unmarshal(body, &value); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return err
	}
	for _, name := range []string{"type", "text", "thinking", "signature", "data", "id", "name", "input", "tool_use_id", "content", "source"} {
		delete(fields, name)
	}
	*b = Block(value)
	if len(fields) > 0 {
		b.extra = fields
	}
	return nil
}

func (b Block) MarshalJSON() ([]byte, error) {
	type plain Block
	var value any = plain(b)
	switch b.Type {
	case "thinking":
		value = struct {
			plain
			Thinking string `json:"thinking"`
		}{plain(b), b.Thinking}
	case "text":
		value = struct {
			plain
			Text string `json:"text"`
		}{plain(b), b.Text}
	case "tool_result":
		value = struct {
			plain
			Content string `json:"content"`
		}{plain(b), b.Content}
	}
	body, err := json.Marshal(value)
	if err != nil || len(b.extra) == 0 {
		return body, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, err
	}
	for name, value := range b.extra {
		fields[name] = value
	}
	return json.Marshal(fields)
}
