// Purpose: Preserve source mutation detection, image ordering and ownership in native DTO reuse.
// Pattern: Differential source-to-wire tests against the uncached adapter.
// Usage: go test ./internal/provider/protocol -run TestCloudSources.
// Related: internal/provider/protocol/cloud_sources.go, internal/provider/protocol/tools.go.
package protocol

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func checkCloudSources(t *testing.T, encoder *Encoder, kind connection.ProviderKind, source []modelapi.Message) Body {
	t.Helper()
	request := modelapi.Request{ModelID: "model", MaxOutput: 32}
	plain, err := Messages(kind, source)
	if err != nil {
		t.Fatal(err)
	}
	_, want, err := Encode(kind, "https://model.test", nil, request, plain, Controls{})
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := encoder.Messages(kind, request.ModelID, source)
	if err != nil {
		t.Fatal(err)
	}
	if encoder.cloudRetained+encoder.cloudSourceBytes > encoder.limit() {
		t.Fatal("source preparation exceeded the shared cache budget")
	}
	_, body, err := encoder.Encode(kind, "https://model.test", nil, request, prepared, Controls{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(body.Reader())
	if err != nil {
		t.Fatal(err)
	}
	decode := func(raw []byte) any {
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	if !reflect.DeepEqual(decode(got), decode(want)) {
		t.Fatal("prepared source differs from current uncached history")
	}
	if encoder.cloudRetained+encoder.cloudSourceBytes > encoder.limit() {
		t.Fatal("native encoding exceeded the shared cache budget")
	}
	return body
}

func TestCloudSourcesDetectBorrowedArgumentChanges(t *testing.T) {
	for _, kind := range []connection.ProviderKind{connection.Claude, connection.Gemini} {
		var encoder Encoder
		arguments := json.RawMessage(`{"content":"` + strings.Repeat("x", 128<<10) + `"}`)
		source := []modelapi.Message{{Role: "assistant", Calls: []toolapi.Call{{ID: "call", Name: "write_file", Arguments: arguments}}}, {Role: "tool", ToolCallID: "call", ToolName: "write_file", Content: "done"}}
		previous := checkCloudSources(t, &encoder, kind, source)
		before, err := io.ReadAll(previous.Reader())
		if err != nil {
			t.Fatal(err)
		}
		checkCloudSources(t, &encoder, kind, source)
		arguments[20] = 'y'
		checkCloudSources(t, &encoder, kind, source)
		after, err := io.ReadAll(previous.Reader())
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("source mutation changed a replayable request", err)
		}
		arguments[0] = '!'
		if _, err := encoder.Messages(kind, "model", source); err == nil || encoder.cloudGroups != nil || encoder.cloud != nil {
			t.Fatal("invalid source bypassed validation or retained native state", err)
		}
	}
}

func TestCloudSourcesPreserveToolImageGroups(t *testing.T) {
	image, err := media.New("fixture.png", "image/png", "png", 1, 1, []byte("fixture"))
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []connection.ProviderKind{connection.Claude, connection.Gemini} {
		encoder := NewEncoder(256 << 10)
		source := []modelapi.Message{
			{Role: "system", Content: strings.Repeat("rules ", 12000)},
			{Role: "assistant", Calls: []toolapi.Call{{ID: "a", Name: "read_file", Arguments: json.RawMessage(`{}`)}, {ID: "b", Name: "read_file", Arguments: json.RawMessage(`{}`)}}},
			{Role: "tool", ToolCallID: "a", ToolName: "read_file", Content: "first", Images: []media.Image{image}},
			{Role: "tool", ToolCallID: "b", ToolName: "read_file", Content: "second"},
			{Role: "user", Content: "continue"},
		}
		for range 2 {
			checkCloudSources(t, &encoder, kind, source)
		}
		source[2].Content = "changed observation"
		checkCloudSources(t, &encoder, kind, source)
		source = source[:2]
		checkCloudSources(t, &encoder, kind, source)
		encoder.Clear()
		if encoder.cloudGroups != nil || encoder.cloud != nil || encoder.cloudSourceBytes != 0 {
			t.Fatal("cleanup retained native source groups")
		}
	}
}
