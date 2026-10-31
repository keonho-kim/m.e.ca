// Purpose: Reuse per-message wire fragments for exact chat tokenization without a full body copy.
// Pattern: Replayable segmented request value.
// Usage: vLLM preflight reads TokenizerBody before the same messages enter Encoder.Encode.
// Related: internal/provider/protocol/body.go, internal/provider/httpclient/context.go.
package protocol

func TokenizerBody(messages []Message, header any) (Body, error) {
	encoded, err := wireJSON.Marshal(header)
	if err != nil {
		return Body{}, err
	}
	encoded = append(encoded[:len(encoded)-1], `,"messages":[`...)
	parts := make([][]byte, 0, 2*len(messages)+2)
	parts = append(parts, encoded)
	size := len(encoded) + 2
	for i := range messages {
		if i > 0 {
			parts = append(parts, []byte(","))
			size++
		}
		content := messages[i].encoded
		if content == nil {
			content, err = wireJSON.Marshal(messages[i])
			if err != nil {
				return Body{}, err
			}
			messages[i].encoded = content
		}
		parts = append(parts, content)
		size += len(content)
	}
	return Body{parts: append(parts, []byte("]}")), size: size}, nil
}
