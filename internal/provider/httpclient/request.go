// Purpose: Translate agent generation requests into native and compatible wire formats.
// Pattern: Adapter.
// Usage: Client.Generate and connection diagnostics.
// Related: internal/agent/model/model.go, internal/provider/protocol/stream.go, internal/provider/protocol/native/frame.go.

package httpclient

import (
	"context"
	"errors"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/protocol"
	"github.com/keonho-kim/meca/internal/provider/transport"
	"io"
	"net/http"
	"slices"
)

func (client *Client) Generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	return client.generate(ctx, request, emit, false, nil)
}

func (client *Client) generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error, probe bool, encoder *protocol.Encoder) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	hasImages, err := imageMessages(request.Messages)
	if err != nil {
		return err
	}
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
	if model := client.summaryUnsupportedModel.Load(); model != nil && *model == request.ModelID {
		controls.OmitThinkingSummary = true
	}
	history := slices.Clone(request.Messages)
	for i := range history {
		if history[i].ProviderID != client.connection.ID || history[i].ModelID != request.ModelID {
			history[i].ReasoningState = nil
		}
	}
	messages, err := encoder.Messages(client.connection.Kind, request.ModelID, history)
	if err != nil {
		return err
	}
	// The probe exercises image acceptance directly, without requiring a separate
	// tokenizer endpoint that may be unavailable or unable to process media.
	if !probe {
		request.MaxOutput, err = client.contextOutput(ctx, request.ModelID, messages, controls, request.MaxOutput, request.Tools, encoder)
	}
	if err != nil {
		return err
	}
	// The acceptance probe leaves the server's loaded context configuration alone.
	if probe {
		info = nil
	}
	endpoint, content, err := encoder.Encode(client.connection.Kind, client.connection.APIBase(), info, request, messages, controls)
	if err != nil {
		return err
	}
	if content.Len() > modelapi.MaxResponseBytes {
		return fmt.Errorf("model context exceeds request size limit")
	}
	if !hasImages {
		if err := emit(modelapi.Event{InputBytes: int64(content.Len())}); err != nil {
			return err
		}
	}
	idle := request.IdleTimeout
	if idle <= 0 {
		idle = transport.DefaultIdleTimeout
	}
	return client.transport.Attempts(ctx, func() (err error) {
		for {
			httpRequest, buildErr := client.transport.NewRequest(ctx, http.MethodPost, endpoint, content.Reader())
			if buildErr != nil {
				return &transport.ConsumerError{Cause: buildErr}
			}
			if httpRequest.GetBody == nil {
				httpRequest.ContentLength = int64(content.Len())
				httpRequest.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(content.Reader()), nil }
			}
			response, sendErr := client.transport.SendOnce(httpRequest, idle)
			if sendErr != nil {
				if !controls.OmitThinkingSummary && transport.ThinkingSummaryUnsupported(sendErr, client.connection.Kind) {
					// A permanent 400 rejects the option before generation. Observe this
					// model's capability once without changing explicit reasoning effort.
					controls.OmitThinkingSummary = true
					client.summaryUnsupportedModel.Store(new(request.ModelID))
					// Rebuild only on this rejected option. Keeping converted DTOs
					// in the attempt closure would pin their base64 images and native
					// histories throughout a potentially long streaming response.
					messages, convertErr := encoder.Messages(client.connection.Kind, request.ModelID, history)
					if convertErr != nil {
						return &transport.ConsumerError{Cause: convertErr}
					}
					endpoint, content, err = encoder.Encode(client.connection.Kind, client.connection.APIBase(), info, request, messages, controls)
					if err != nil {
						return &transport.ConsumerError{Cause: err}
					}
					if !hasImages {
						if err := emit(modelapi.Event{InputBytes: int64(content.Len())}); err != nil {
							return &transport.ConsumerError{Cause: err}
						}
					}
					continue
				}
				return sendErr
			}
			defer func() { err = errors.Join(err, response.Body.Close()) }()
			return protocol.ReadResponse(ctx, response, client.connection.Kind, func(event modelapi.Event) error {
				if event.Done {
					var err error
					event.ReasoningState, err = content.BindContinuation(event.ReasoningState)
					if err != nil {
						return &transport.ConsumerError{Cause: err}
					}
				}
				if err := emit(event); err != nil {
					return &transport.ConsumerError{Cause: err}
				}
				return nil
			})
		}
	}, func(retry modelapi.Retry) error { return emit(modelapi.Event{Retry: &retry}) })
}
