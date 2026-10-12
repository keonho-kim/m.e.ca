// Purpose: Discover verified boolean Thinking controls without guessing provider parameters.
// Pattern: HTTP Adapter.
// Usage: Called when selecting or restoring an Ollama model.
// Related: internal/provider/httpclient/request.go, internal/provider/httpclient/thinking_test.go.
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
)

func (c *Client) ThinkingControl(ctx context.Context, model string) (supported bool, err error) {
	if c.connection.Kind != modelconn.Ollama {
		return false, nil
	}
	payload, err := json.Marshal(map[string]string{"model": model})
	if err != nil {
		return false, err
	}
	response, err := c.do(ctx, http.MethodPost, c.url("show"), bytes.NewReader(payload))
	if err != nil {
		return false, err
	}
	defer func() { err = errors.Join(err, response.Body.Close()) }()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024+1))
	if err != nil {
		return false, err
	}
	if len(body) > 1024*1024 {
		return false, fmt.Errorf("model metadata exceeds limit")
	}
	var info struct {
		Thinking struct {
			Values []json.RawMessage `json:"values"`
		} `json:"thinking"`
	}
	if err = json.Unmarshal(body, &info); err != nil {
		return false, err
	}
	on, off := false, false
	for _, v := range info.Thinking.Values {
		if string(v) == "true" {
			on = true
		}
		if string(v) == "false" {
			off = true
		}
	}
	return on && off, nil
}
