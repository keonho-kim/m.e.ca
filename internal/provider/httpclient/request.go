// Purpose: Translate agent generation requests into native and compatible wire formats.
// Pattern: Adapter.
// Usage: Client.Generate and connection diagnostics.
// Related: internal/agent/model/model.go, internal/provider/protocol/stream.go, internal/provider/protocol/wire.go.

package httpclient

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/provider/protocol"
	"github.com/keonho-kim/v-code/internal/provider/transport"
	"net/http"
)

func (client *Client) Generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) (err error) {
	if err := modelapi.CheckTemperature(request.Temperature); err != nil {
		return err
	}
	if emit == nil {
		return fmt.Errorf("a model event consumer is required")
	}
	if request.ModelID == "" || len(request.ModelID) > 512 || request.MaxOutput < 1 || request.MaxOutput > 65536 || request.IdleTimeout < 0 {
		return fmt.Errorf("invalid model request limits")
	}
	if len(request.Messages) == 0 {
		return fmt.Errorf("invalid message or tool count")
	}
	if client.modelInfo != nil {
		if client.modelInfo.MaxOutput > 0 {
			request.MaxOutput = min(request.MaxOutput, client.modelInfo.MaxOutput)
		}
		if client.modelInfo.ContextWindow > 0 {
			request.MaxOutput = min(request.MaxOutput, client.modelInfo.ContextWindow-1)
		}
		if request.MaxOutput < 1 {
			return fmt.Errorf("model context has no output capacity")
		}
	}
	controls, err := protocol.ControlsFor(client.connection.Kind, client.profileID, client.modelInfo, request)
	if err != nil {
		return err
	}
	messages := make([]protocol.Message, 0, len(request.Messages))
	for _, message := range request.Messages {
		if message.Role != "system" && message.Role != "user" && message.Role != "assistant" && message.Role != "tool" {
			return fmt.Errorf("invalid message role")
		}

		if len(message.Calls) > 0 || message.Role == "tool" {
			return fmt.Errorf("model context must project tool audit records as text observations")
		}
		messages = append(messages, protocol.Message{Role: message.Role, Content: message.Content})
	}
	request.MaxOutput, err = client.contextOutput(ctx, request.ModelID, messages, controls, request.MaxOutput)
	if err != nil {
		return err
	}
	endpoint, content, err := protocol.Encode(client.connection.Kind, client.connection.APIBase(), client.modelInfo, request, messages, controls)
	if err != nil {
		return err
	}
	if len(content) > 4*1024*1024 {
		return fmt.Errorf("model context exceeds request size limit")
	}
	idle := request.IdleTimeout
	if idle <= 0 {
		idle = transport.DefaultIdleTimeout
	}
	return client.transport.Attempts(ctx, func() (err error) {
		request, err := client.transport.NewRequest(ctx, http.MethodPost, endpoint, bytes.NewReader(content))
		if err != nil {
			return &transport.ConsumerError{Cause: err}
		}
		response, err := client.transport.SendOnce(request, idle)
		if err != nil {
			return err
		}
		defer func() { err = errors.Join(err, response.Body.Close()) }()
		return protocol.ReadResponse(ctx, response, client.connection.Kind, func(event modelapi.Event) error {
			if err := emit(event); err != nil {
				return &transport.ConsumerError{Cause: err}
			}
			return nil
		})
	}, func(retry modelapi.Retry) error { return emit(modelapi.Event{Retry: &retry}) })
}
