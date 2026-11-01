// Purpose: Own the selected connection, active settings form and adopted client lifetime.
// Pattern: Event-loop-owned connection state.
// Usage: Session commits settings before adoption and releases clients during shutdown.
// Related: internal/tui/session/connection.go, internal/tui/session/model_form.go, internal/tui/session/app.go.
package session

import (
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
	modelsettings "github.com/keonho-kim/meca/internal/tui/settings/model"
)

type connectionState struct {
	setupRevision int
	modelForm     *modelsettings.Form
	selection     modelconn.Selection
	client        *httpclient.Client
}

func (c *connectionState) adopt(client *httpclient.Client, selection modelconn.Selection) {
	previous := c.client
	c.client, c.selection = client, selection
	if previous != nil && previous != client {
		previous.Close()
	}
}

func (c *connectionState) close() {
	if c.client != nil {
		c.client.Close()
		c.client = nil
	}
}

func (c *connectionState) rejectImages() {
	if c.selection.Details != nil {
		info := *c.selection.Details
		info.ImageInput.State = "unsupported"
		info.ImageInput.Source = "provider rejection"
		c.selection.Details = &info
	}
}
