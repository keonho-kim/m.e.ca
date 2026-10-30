// Purpose: Isolate minimal image probes and reject unconfirmed image inputs during normal generation.
// Pattern: Connection-bound image request boundary.
// Usage: Setup probes with no task context; Generate enforces the established capability.
// Related: internal/provider/httpclient/request.go, internal/provider/diagnostic/image.go.
package httpclient

import (
	"context"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

// ProbeImage sends only a minimal self-contained connection test.
func (c *Client) ProbeImage(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	if len(request.Messages) != 1 || request.Messages[0].Role != "user" || len(request.Messages[0].Images) != 1 || len(request.Tools) != 0 || request.MaxOutput < 1 || request.MaxOutput > 32 || len(request.Messages[0].Images[0].Data) > 4096 {
		return fmt.Errorf("invalid image-input probe")
	}
	return c.generate(ctx, request, emit, true)
}

// ImageInputSupported includes definitive rejection observed since the model was bound.
func (c *Client) ImageInputSupported() bool {
	return c.modelInfo != nil && c.modelInfo.ImageInput.Supported() && !c.imageRejected.Load()
}

func imageMessages(messages []modelapi.Message) bool {
	for _, message := range messages {
		if len(message.Images) > 0 {
			return true
		}
	}
	return false
}
