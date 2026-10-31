// Purpose: Verify segmented tokenization preserves exact messages, lengths and replay cursors.
// Pattern: Differential wire-format tests.
// Usage: go test ./internal/provider/protocol -run TestTokenizerBody.
// Related: internal/provider/protocol/tokenizer_body.go, internal/provider/protocol/source_history.go.
package protocol

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestTokenizerBodyPreservesCachedAndChangedNativeHistory(t *testing.T) {
	var encoder Encoder
	defer encoder.Clear()
	source := []modelapi.Message{{Role: "assistant", Calls: []toolapi.Call{{ID: "one", Name: "write_file", Arguments: json.RawMessage(`{"path":"x","content":"` + strings.Repeat("x", 1<<20) + `"}`)}}}}
	header := struct {
		Model string `json:"model"`
	}{"model"}
	for step := range 3 {
		if step == 2 {
			source = append(source, modelapi.Message{Role: "tool", ToolCallID: "one", ToolName: "write_file", Content: "observed result"})
		}
		plain, err := Messages(connection.VLLM, source)
		if err != nil {
			t.Fatal(err)
		}
		want, err := wireJSON.Marshal(struct {
			Model    string    `json:"model"`
			Messages []Message `json:"messages"`
		}{header.Model, plain})
		if err != nil {
			t.Fatal(err)
		}
		messages, err := encoder.Messages(connection.VLLM, header.Model, source)
		if err != nil {
			t.Fatal(err)
		}
		body, err := TokenizerBody(messages, header)
		if err != nil {
			t.Fatal(err)
		}
		first, err := io.ReadAll(body.Reader())
		if err != nil || len(first) != body.Len() {
			t.Fatal("invalid tokenizer body length", err)
		}
		var a, b any
		if json.Unmarshal(want, &a) != nil || json.Unmarshal(first, &b) != nil || !reflect.DeepEqual(a, b) {
			t.Fatal("tokenizer request changed")
		}
		if _, _, err = encoder.Encode(connection.VLLM, "https://model.test", nil, modelapi.Request{ModelID: header.Model, MaxOutput: 12}, messages, Controls{}); err != nil {
			t.Fatal(err)
		}
		reader := body.Reader().(*bodyReader)
		if _, err = reader.Read(make([]byte, 17)); err != nil {
			t.Fatal(err)
		}
		again, err := io.ReadAll(reader.Replay())
		if err != nil || !bytes.Equal(first, again) {
			t.Fatal("tokenizer retry reused a consumed cursor", err)
		}
	}
}
