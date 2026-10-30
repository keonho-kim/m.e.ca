// Purpose: Bind checked model metadata before concurrent generation begins.
// Pattern: Simple Module.
// Usage: Setup discovers and diagnoses a model before transferring the client.
// Related: internal/provider/connection/models.go, internal/provider/setup/setup.go.
package httpclient

import "github.com/keonho-kim/meca/internal/provider/connection"

func (c *Client) BindModel(info connection.ModelInfo) error {
	if err := info.Check(); err != nil {
		return err
	}
	c.imageRejected.Store(false)
	c.modelInfo = &info
	return nil
}
