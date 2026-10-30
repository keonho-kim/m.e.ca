// Purpose: Resolve one selected model's image capability before client ownership transfers.
// Pattern: Connection preparation step.
// Usage: Prepare reuses metadata and runs at most one probe for unknown non-OpenRouter models.
// Related: internal/provider/diagnostic/image.go, internal/provider/connection/image.go.
package setup

import (
	"context"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/diagnostic"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
)

func prepareImage(ctx context.Context, client *httpclient.Client, kind connection.ProviderKind, info *connection.ModelInfo) error {
	if info.ImageInput.Known() || kind == connection.OpenRouter {
		return nil
	}
	value, err := diagnostic.CheckImage(ctx, client.ProbeImage, info.ID)
	if err != nil {
		return err
	}
	info.ImageInput = value
	return client.BindModel(*info)
}
