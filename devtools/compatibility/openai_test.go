// Purpose: Verify the official OpenAI Go SDK against MECA's local compatible endpoints.
// Pattern: External client contract test with an injected local server address.
// Usage: internal/serverapi starts a mock-backed server and runs this separate development module.
// Related: internal/serverapi/openai_sdk_test.go, devtools/go.mod.
package compatibility

import (
	"os"
	"strings"
	"testing"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

func TestOfficialGoSDK(t *testing.T) {
	base := os.Getenv("MECA_COMPAT_BASE_URL")
	if base == "" {
		t.Skip("run through internal/serverapi's mock-server fixture")
	}
	client := openai.NewClient(option.WithBaseURL(base), option.WithAPIKey(os.Getenv("MECA_COMPAT_TOKEN")), option.WithMaxRetries(0))
	models, err := client.Models.List(t.Context())
	if err != nil || len(models.Data) != 1 || models.Data[0].ID != "meca" {
		t.Fatal(models, err)
	}
	model, err := client.Models.Get(t.Context(), "meca")
	if err != nil || model.ID != "meca" {
		t.Fatal(model, err)
	}
	chat := openai.ChatCompletionNewParams{Model: "meca", Messages: []openai.ChatCompletionMessageParamUnion{openai.UserMessage("hello")}}
	completion, err := client.Chat.Completions.New(t.Context(), chat)
	if err != nil || len(completion.Choices) != 1 || completion.Choices[0].Message.Content != "done" || completion.Choices[0].FinishReason != "stop" {
		t.Fatal(completion, err)
	}
	stream := client.Chat.Completions.NewStreaming(t.Context(), chat)
	var text strings.Builder
	for stream.Next() {
		for _, choice := range stream.Current().Choices {
			text.WriteString(choice.Delta.Content)
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	_ = stream.Close()
	if text.String() != "done" {
		t.Fatal(text.String())
	}
	request := responses.ResponseNewParams{Model: "meca", Input: responses.ResponseNewParamsInputUnion{OfString: openai.String("hello")}}
	response, err := client.Responses.New(t.Context(), request)
	if err != nil || response.OutputText() != "done" || response.Status != "completed" {
		t.Fatal(response, err)
	}
	stored, err := client.Responses.Get(t.Context(), response.ID, responses.ResponseGetParams{})
	if err != nil || stored.OutputText() != "done" {
		t.Fatal(stored, err)
	}
	responseStream := client.Responses.NewStreaming(t.Context(), request)
	text.Reset()
	completed := false
	for responseStream.Next() {
		event := responseStream.Current()
		if event.Type == "response.output_text.delta" {
			text.WriteString(event.AsResponseOutputTextDelta().Delta)
		}
		if event.Type == "response.completed" {
			completed = true
			if event.AsResponseCompleted().Response.OutputText() != "done" {
				t.Fatal(event)
			}
		}
	}
	if err := responseStream.Err(); err != nil {
		t.Fatal(err)
	}
	_ = responseStream.Close()
	if !completed || text.String() != "done" {
		t.Fatal(completed, text.String())
	}
	request.Background = openai.Bool(true)
	background, err := client.Responses.New(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Responses.Cancel(t.Context(), background.ID); err != nil {
		t.Fatal(err)
	}
}
