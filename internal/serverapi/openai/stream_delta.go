// Purpose: Encode the frequent Responses delta without per-event maps and stable-ID rebuilding.
// Pattern: Typed wire projection.
// Usage: encoder.delta supplies the public item identity and monotonically increasing event sequence.
// Related: internal/serverapi/openai/stream.go.
package openai

type textDelta struct {
	Type     string `json:"type"`
	Sequence int    `json:"sequence_number"`
	Item     string `json:"item_id"`
	Output   int    `json:"output_index"`
	Content  int    `json:"content_index"`
	Delta    string `json:"delta"`
	Logprobs []any  `json:"logprobs"`
}
