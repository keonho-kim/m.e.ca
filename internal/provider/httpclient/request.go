// Purpose: Prepare an immutable provider request and measure its effective context before generation.
// Pattern: HTTP adapter preparation boundary.
// Usage: Direct generation and the agent's run-scoped generator share preparation.
// Related: internal/provider/httpclient/context.go, internal/provider/httpclient/generation.go, internal/provider/protocol/encoder.go.
package httpclient

import (
	"context"
	"slices"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/protocol"
)

type preparedRequest struct {
	request          modelapi.Request
	budget           modelapi.InputBudget
	messages         []protocol.Message
	controls         protocol.Controls
	encoder          *protocol.Encoder
	info             *connection.ModelInfo
	probe, hasImages bool
}

func (client *Client) Generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	return client.generate(ctx, request, emit, false, nil)
}

func (client *Client) generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error, probe bool, encoder *protocol.Encoder) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if emit == nil {
		return faults.Errorf(diagcatalog.HttpclientAModelEventConsumerIsRequired)
	}
	prepared, err := client.prepare(ctx, request, probe, encoder)
	if err != nil {
		return err
	}
	return prepared.Send(ctx, emit)
}

func (client *Client) prepare(ctx context.Context, request modelapi.Request, probe bool, encoder *protocol.Encoder) (modelapi.PreparedGeneration, error) {
	if err := ctx.Err(); err != nil {
		return modelapi.PreparedGeneration{}, err
	}
	hasImages, err := imageMessages(request.Messages)
	if err != nil {
		return modelapi.PreparedGeneration{}, &modelapi.ContextLimitError{Cause: err}
	}
	if hasImages && !probe && !client.ImageInputSupported() {
		return modelapi.PreparedGeneration{}, &modelapi.ImageInputError{}
	}
	if err := modelapi.CheckTemperature(request.Temperature); err != nil {
		return modelapi.PreparedGeneration{}, err
	}
	if request.ModelID == "" || len(request.ModelID) > 512 || request.IdleTimeout < 0 {
		return modelapi.PreparedGeneration{}, faults.Errorf(diagcatalog.HttpclientInvalidModelRequestLimits)
	}
	if len(request.Messages) == 0 {
		return modelapi.PreparedGeneration{}, faults.Errorf(diagcatalog.HttpclientInvalidMessageOrToolCount)
	}
	if err := modelapi.CheckOutputBudget(request.MaxOutput); err != nil {
		return modelapi.PreparedGeneration{}, err
	}
	controls, err := protocol.ControlsFor(client.connection.Kind, client.modelInfo, request)
	if err != nil {
		return modelapi.PreparedGeneration{}, err
	}
	if model := client.summaryUnsupportedModel.Load(); model != nil && *model == request.ModelID {
		controls.OmitThinkingSummary = true
	}
	request.Messages = slices.Clone(request.Messages)
	for i := range request.Messages {
		if request.Messages[i].ProviderID != client.connection.ID || request.Messages[i].ModelID != request.ModelID {
			request.Messages[i].ReasoningState = nil
		}
	}
	messages, err := encoder.Messages(client.connection.Kind, request.ModelID, request.Messages)
	if err != nil {
		return modelapi.PreparedGeneration{}, err
	}
	prepared := preparedRequest{request: request, messages: messages, controls: controls, encoder: encoder, probe: probe, hasImages: hasImages}
	// The acceptance probe leaves the loaded context configuration alone.
	if !probe {
		prepared.budget, err = client.contextOutput(ctx, request, messages, controls, encoder)
		if err != nil {
			return modelapi.PreparedGeneration{}, err
		}
		prepared.request.MaxOutput = prepared.budget.OutputTokens
		prepared.info = client.effectiveModel()
	}
	return modelapi.PreparedGeneration{
		Budget: prepared.budget,
		Send: func(ctx context.Context, emit func(modelapi.Event) error) error {
			return client.sendPrepared(ctx, prepared, emit)
		},
	}, nil
}
