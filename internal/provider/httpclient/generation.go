// Purpose: Send one prepared provider request and own its response, retries and continuation.
// Pattern: Serial HTTP generation boundary.
// Usage: Client.prepare binds a measured request to sendPrepared.
// Related: internal/provider/httpclient/request.go, internal/provider/protocol/stream.go, internal/provider/transport/retry.go.
package httpclient

import (
	"context"
	"errors"
	"io"
	"net/http"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/protocol"
	"github.com/keonho-kim/meca/internal/provider/transport"
)

func (client *Client) sendPrepared(ctx context.Context, prepared preparedRequest, emit func(modelapi.Event) error) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	defer func() {
		if prepared.hasImages && !prepared.probe && transport.ImageUnsupported(err) {
			client.imageRejected.Store(true)
			err = &modelapi.ImageInputError{Cause: err}
		}
	}()
	kind, base := client.connection.Kind, client.connection.APIBase()
	endpoint, content, err := prepared.encode(kind, base)
	if err != nil {
		return err
	}
	if !prepared.hasImages {
		if err := emit(modelapi.Event{InputBytes: int64(content.Len()), InputBudget: &prepared.budget}); err != nil {
			return err
		}
	}
	idle := prepared.request.IdleTimeout
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
				if !prepared.probe {
					if capacity := transport.OutputCapacityError(sendErr, prepared.request.MaxOutput); capacity != nil {
						return capacity
					}
				}
				if prepared.controls.OmitThinkingSummary || !transport.ThinkingSummaryUnsupported(sendErr, kind) {
					return sendErr
				}
				prepared.controls.OmitThinkingSummary = true
				client.summaryUnsupportedModel.Store(new(prepared.request.ModelID))
				// Rebuild the same immutable history only after confirmed rejection.
				prepared.messages, err = prepared.encoder.Messages(kind, prepared.request.ModelID, prepared.request.Messages)
				if err != nil {
					return &transport.ConsumerError{Cause: err}
				}
				endpoint, content, err = prepared.encode(kind, base)
				if err != nil {
					return &transport.ConsumerError{Cause: err}
				}
				if !prepared.hasImages {
					if err := emit(modelapi.Event{InputBytes: int64(content.Len())}); err != nil {
						return &transport.ConsumerError{Cause: err}
					}
				}
				continue
			}
			// The decoder owns the outcome. Closing the read-only body releases
			// the connection, including after an accepted completion.
			defer response.Body.Close()
			return protocol.ReadResponse(attemptCtx, response, kind, func(event modelapi.Event) error {
				output := event.HasOutput()
				if event.Progress || output {
					generationStarted = true
					progress.Activity()
				}
				if event.Progress && !output && event.Usage == nil {
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

func (prepared preparedRequest) encode(kind connection.ProviderKind, base string) (string, protocol.Body, error) {
	endpoint, content, err := prepared.encoder.Encode(kind, base, prepared.info, prepared.request, prepared.messages, prepared.controls)
	if err != nil {
		return "", protocol.Body{}, err
	}
	if content.Len() > modelapi.MaxResponseBytes {
		return "", protocol.Body{}, &modelapi.ContextLimitError{Cause: faults.Errorf(diagcatalog.HttpclientModelContextExceedsRequestSizeLimit)}
	}
	return endpoint, content, nil
}
