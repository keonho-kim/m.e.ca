// Purpose: Check image request acceptance without evaluating visual recognition or exposing user content.
// Pattern: Bounded connection diagnostic.
// Usage: Setup calls CheckImage only when model metadata does not determine support.
// Related: internal/provider/setup/image.go, internal/provider/httpclient/image.go.
package diagnostic

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"time"

	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/transport"
)

// ImageGenerator sends the probe through the same transport as normal generation.
type ImageGenerator func(context.Context, modelapi.Request, func(modelapi.Event) error) error

// The fixture is encoded once; each request borrows its immutable compressed bytes.
var probeImage = func() media.Image {
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewGray(image.Rect(0, 0, 16, 16))); err != nil {
		panic(err)
	}
	value, err := media.New("connection-image-probe.png", "image/png", "png", 16, 16, data.Bytes())
	if err != nil {
		panic(err)
	}
	return value
}()

// CheckImage tests request acceptance without judging the generated answer.
func CheckImage(ctx context.Context, generate ImageGenerator, model string) (connection.ImageInput, error) {
	if generate == nil {
		return connection.ImageInput{}, fmt.Errorf("image probe generator is required")
	}
	parent := ctx
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	accepted, received := false, false
	request := modelapi.Request{ModelID: model, MaxOutput: 32, IdleTimeout: 15 * time.Second, Messages: []modelapi.Message{{Role: "user", Content: "Reply briefly.", Images: []media.Image{probeImage}}}}
	err := generate(ctx, request, func(event modelapi.Event) error {
		if event.Retry != nil {
			accepted, received = false, false
		}
		accepted = accepted || event.Done
		received = received || event.Delta != "" || event.Thinking != "" || event.Usage != nil
		return ctx.Err()
	})
	if parent.Err() != nil {
		return connection.ImageInput{}, parent.Err()
	}
	value := connection.ImageInput{State: "unknown", Source: "image input probe"}
	if err != nil {
		value.Detail = err.Error()
	}
	var limited *modelapi.OutputLimitError
	if err == nil && accepted || errors.As(err, &limited) && received {
		value.State = "supported"
		value.Detail = ""
	} else if transport.ImageUnsupported(err) {
		value.State = "unsupported"
	}
	return value, nil
}
