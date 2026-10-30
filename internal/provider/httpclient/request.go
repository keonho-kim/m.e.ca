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
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/protocol"
	"github.com/keonho-kim/meca/internal/provider/transport"
	"net/http"
	"slices"
)

func (client *Client) Generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	return client.generate(ctx, request, emit, false)
}

func (client *Client) generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error, probe bool) (err error) {
	hasImages := imageMessages(request.Messages)
	if hasImages && !probe && !client.ImageInputSupported() {
		return &modelapi.ImageInputError{}
	}
	defer func() {
		if hasImages && !probe && transport.ImageUnsupported(err) {
			client.imageRejected.Store(true)
			err = &modelapi.ImageInputError{Cause: err}
		}
	}()
	if err := modelapi.CheckTemperature(request.Temperature); err != nil {
		return err
	}
	if emit == nil {
		return fmt.Errorf("a model event consumer is required")
	}
	if request.ModelID == "" || len(request.ModelID) > 512 || request.IdleTimeout < 0 {
		return fmt.Errorf("invalid model request limits")
	}
	if len(request.Messages) == 0 {
		return fmt.Errorf("invalid message or tool count")
	}
	if err := modelapi.CheckOutputBudget(request.MaxOutput); err != nil {
		return err
	}
	info := client.effectiveModel()
	controls, err := protocol.ControlsFor(client.connection.Kind, client.modelInfo, request)
	if err != nil {
		return err
	}
	history := slices.Clone(request.Messages)
	for i := range history {
		if history[i].ProviderID != client.connection.ID || history[i].ModelID != request.ModelID {
			history[i].ReasoningState = nil
		}
	}
	messages, err := protocol.Messages(client.connection.Kind, history)
	if err != nil {
		return err
	}
	// The probe exercises image acceptance directly, without requiring a separate
	// tokenizer endpoint that may be unavailable or unable to process media.
	if !probe {
		request.MaxOutput, err = client.contextOutput(ctx, request.ModelID, messages, controls, request.MaxOutput, request.Tools)
	}
	if err != nil {
		return err
	}
	// The acceptance probe leaves the server's loaded context configuration alone.
	if probe {
		info = nil
	}
	endpoint, content, err := protocol.Encode(client.connection.Kind, client.connection.APIBase(), info, request, messages, controls)
	if err != nil {
		return err
	}
	if len(content) > modelapi.MaxResponseBytes {
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
