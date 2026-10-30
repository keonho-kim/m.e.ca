// Purpose: Preserve immutable tool images in vision context and explicitly omit them for text models.
// Pattern: Pure context projection regression tests.
// Usage: go test ./internal/agent/context -run TestImage.
// Related: internal/agent/context/request.go, internal/agent/context/conversation.go.
package context

import (
	"bytes"
	"encoding/json"
	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
	"testing"
)

func TestImageProjectionAndModelChange(t *testing.T) {
	picture, _ := media.New("shot.png", "image/png", "png", 8, 8, []byte("image payload"))
	call := toolapi.Call{ID: "read", Name: "read_image", Arguments: json.RawMessage(`{"path":"shot.png"}`)}
	messages := []modelapi.Message{{Role: "user", Content: "Inspect"}, {Role: "assistant", Calls: []toolapi.Call{call}}, {Role: "tool", ToolName: call.Name, ToolCallID: call.ID, Content: "Image loaded", Images: []media.Image{picture}}}
	var projection Projection
	input := Input{Request: modelapi.Request{ModelID: "vision"}, ImageInput: true, System: "Rules", Messages: messages, MaxBytes: 16 * 1024 * 1024, Redact: func(s string) string { return s }, Projection: &projection, RedactionVersion: func() int { return 1 }}
	for _, enabled := range []bool{true, false, true} {
		input.ImageInput = enabled
		request, err := Build(input)
		if err != nil {
			t.Fatal(err)
		}
		got := request.Messages[len(request.Messages)-1]
		if enabled && (len(got.Images) != 1 || !bytes.Equal(got.Images[0].Data, picture.Data)) {
			t.Fatal("image snapshot lost")
		}
		if !enabled && (len(got.Images) != 0 || !strings.Contains(got.Content, "not provided")) {
			t.Fatal("text model received image or omission was hidden")
		}
	}
	if len(messages[2].Images) != 1 {
		t.Fatal("projection altered original image history")
	}
	// An orphan result remains evidence with pixels, not a dangling native tool result.
	orphan, err := conversationMessage(messages[2], false)
	if err != nil || orphan.Role != "user" || len(orphan.Images) != 1 || strings.Contains(orphan.Content, "aW1hZ2UgcGF5bG9hZA") {
		t.Fatal("orphan image was lost or encoded as text", err)
	}
}

func TestImageBudgetUsesTransmittedContext(t *testing.T) {
	picture, err := media.New("shot.png", "image/png", "png", 32, 32, bytes.Repeat([]byte{42}, 1024*1024))
	if err != nil {
		t.Fatal(err)
	}
	var projection Projection
	input := Input{System: "Rules", Messages: []modelapi.Message{{Role: "user", Content: "Inspect", Images: []media.Image{picture}}}, MaxBytes: 8192, Redact: func(s string) string { return s }, Projection: &projection, RedactionVersion: func() int { return 1 }}
	for _, enabled := range []bool{false, true, false} {
		input.ImageInput = enabled
		request, err := Build(input)
		if enabled {
			if err == nil {
				t.Fatal("vision request exceeded the byte budget without an error")
			}
			continue
		}
		if err != nil || len(request.Messages) != 2 || len(request.Messages[1].Images) != 0 {
			t.Fatalf("omitted image consumed text context budget: %v", err)
		}
		exact := messageBytes(request.Messages[0]) + messageBytes(request.Messages[1])
		input.MaxBytes = exact
		if _, err := Build(input); err != nil {
			t.Fatalf("exact budget rejected: %v", err)
		}
		input.MaxBytes--
		if _, err := Build(input); err == nil {
			t.Fatal("omission notice was not counted")
		}
		input.MaxBytes = 8192
	}
	if len(input.Messages[0].Images) != 1 {
		t.Fatal("budget projection changed the retained image")
	}
}
