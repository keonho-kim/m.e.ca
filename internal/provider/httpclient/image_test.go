// Purpose: Verify definitive image rejection disables future image requests while preserving text generation.
// Pattern: HTTP boundary regression test.
// Usage: go test ./internal/provider/httpclient -run TestImage.
// Related: internal/provider/httpclient/request.go, internal/provider/httpclient/image.go.
package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestImageRejectionDisablesOnlyImageInput(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		var body struct {
			Messages []struct{ Content json.RawMessage }
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if len(body.Messages) > 0 && len(body.Messages[0].Content) > 0 && body.Messages[0].Content[0] == '[' {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "data: {\"error\":{\"code\":\"image_input_not_supported\",\"message\":\"images unavailable\"}}\n\n")
		} else {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"choices":[{"message":{"content":"hello"},"finish_reason":"stop"}]}`)
		}
	}))
	defer server.Close()
	conn, err := connection.New(connection.VLLM, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewClient(conn, "")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.BindModel(connection.ModelInfo{ID: "m", ImageInput: connection.DeclaredImageInput(true, "metadata")}); err != nil {
		t.Fatal(err)
	}
	picture, _ := media.New("shot.png", "image/png", "png", 8, 8, []byte("snapshot"))
	request := modelapi.Request{ModelID: "m", MaxOutput: 32, Messages: []modelapi.Message{{Role: "user", Content: "Inspect", Images: []media.Image{picture}}}}
	for range 2 {
		err := client.Generate(context.Background(), request, func(modelapi.Event) error { return nil })
		var rejection *modelapi.ImageInputError
		if !errors.As(err, &rejection) {
			t.Fatalf("definitive rejection not classified: %v", err)
		}
	}
	if requests.Load() != 1 || client.ImageInputSupported() {
		t.Fatal("rejected image request was repeated")
	}
	request.Messages[0].Images = nil
	if err := client.Generate(context.Background(), request, func(modelapi.Event) error { return nil }); err != nil {
		t.Fatal("text generation blocked", err)
	}
	if requests.Load() != 2 {
		t.Fatal("text request did not execute")
	}
}
