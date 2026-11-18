// Purpose: Translate agent generation requests into native and compatible wire formats.
// Pattern: Adapter.
// Usage: Client.Generate and connection diagnostics.
// Related: internal/agent/model/model.go, internal/provider/protocol/stream.go, internal/provider/protocol/native/frame.go.

package httpclient

import (
	"context"
	"errors"

	"io"
	"net/http"
	"slices"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/protocol"
	"github.com/keonho-kim/meca/internal/provider/transport"
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
		return faults.Errorf(diagcatalog.HttpclientAModelEventConsumerIsRequired)
	}
	if request.ModelID == "" || len(request.ModelID) > 512 || request.IdleTimeout < 0 {
		return faults.Errorf(diagcatalog.HttpclientInvalidModelRequestLimits)
	}
	if len(request.Messages) == 0 {
		return faults.Errorf(diagcatalog.HttpclientInvalidMessageOrToolCount)
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
	canReduce := !probe && request.MaxOutput == 0
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
		return faults.Errorf(diagcatalog.HttpclientModelContextExceedsRequestSizeLimit)
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
	attempt := 0
	generationStarted := false
	err = client.transport.Attempts(ctx, func() (err error) {
		progress := transport.WatchContent(ctx, idle)
		defer progress.Close()
		attemptCtx := progress.Context()
		for {
			attempt++
			httpRequest, buildErr := client.transport.NewRequest(attemptCtx, http.MethodPost, endpoint, content.Reader())
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
					controls.OmitThinkingSummary = true
					client.summaryUnsupportedModel.Store(new(request.ModelID))
				} else {
					if probe {
						return sendErr
					}
					next, failure := rejectedBudget(sendErr, request.MaxOutput, info.ContextWindow, canReduce)
					if failure != nil {
						return failure
					}
					canReduce = false
					reason := diagcatalog.Text(diagcatalog.ProviderReducedOutput, request.MaxOutput, next)
					if err := emit(modelapi.Event{Retry: &modelapi.Retry{Attempt: attempt + 1, MaxAttempts: attempt + 1, Reason: reason}}); err != nil {
						return &transport.ConsumerError{Cause: err}
					}
					request.MaxOutput = next
				}
				// Rebuild the same immutable history only after confirmed rejection.
				retryMessages, err := encoder.Messages(client.connection.Kind, request.ModelID, history)
				if err != nil {
					return &transport.ConsumerError{Cause: err}
				}
				endpoint, content, err = encoder.Encode(client.connection.Kind, client.connection.APIBase(), info, request, retryMessages, controls)
				if err != nil {
					return &transport.ConsumerError{Cause: err}
				}
				if content.Len() > modelapi.MaxResponseBytes {
					return &transport.ConsumerError{Cause: faults.Errorf(diagcatalog.HttpclientModelContextExceedsRequestSizeLimit)}
				}
				if !hasImages {
					if err := emit(modelapi.Event{InputBytes: int64(content.Len())}); err != nil {
						return &transport.ConsumerError{Cause: err}
					}
				}
				continue
			}
			canReduce = false
			defer func() { err = errors.Join(err, response.Body.Close()) }()
			return protocol.ReadResponse(attemptCtx, response, client.connection.Kind, func(event modelapi.Event) error {
				if event.Progress || event.Delta != "" || event.Thinking != "" || len(event.Calls) > 0 || event.Done {
					generationStarted = true
					progress.Activity()
				}
				if event.Progress && event.Delta == "" && event.Thinking == "" && len(event.Calls) == 0 && !event.Done && event.Usage == nil {
					return nil
				}
				if event.Done {
					var err error
					event.ReasoningState, err = content.BindContinuation(event.ReasoningState)
					if err != nil {
						return &transport.ConsumerError{Cause: err}
					}
				}
				if err := progress.Consume(func() error { return emit(event) }); err != nil {
					return &transport.ConsumerError{Cause: err}
				}
				return nil
			})
		}
	}, func(retry modelapi.Retry) error {
		// A failed exchange may have reached the model. Do not start a budget correction after that.
		canReduce = false
		remaining := retry.MaxAttempts - retry.Attempt
		retry.Attempt = attempt + 1
		retry.MaxAttempts = retry.Attempt + remaining
		return emit(modelapi.Event{Retry: &retry})
	})
	var capacity *modelapi.ContextLimitError
	if generationStarted && errors.As(err, &capacity) {
		observed := *capacity
		observed.Partial = true
		observed.Cause = err
		return &observed
	}
	return err
}
