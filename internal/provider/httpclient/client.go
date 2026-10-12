// Purpose: Discover actual server models through provider HTTP adapters.
// Pattern: Adapter.
// Usage: Constructed by cmd/v and injected into model selection and generation.
// Related: internal/provider/connection/connection.go, internal/provider/httpclient/request.go, internal/provider/httpclient/client_test.go.

package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
)

type Client struct {
	connection modelconn.Connection
	key        string
	http       *http.Client
}

func NewClient(connection modelconn.Connection, key string) (*Client, error) {
	parsed, err := modelconn.New(connection.Kind, connection.Endpoint)
	if err != nil {
		return nil, err
	}
	if strings.ContainsAny(key, "\r\n") || len(key) > 16*1024 {
		return nil, fmt.Errorf("invalid API Key")
	}
	parsed.CredentialRef = connection.CredentialRef
	return &Client{connection: parsed, key: key, http: &http.Client{
		Timeout:   3 * time.Minute,
		Transport: &http.Transport{Proxy: http.ProxyFromEnvironment, DialContext: (&net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}).DialContext, MaxIdleConns: 2, MaxIdleConnsPerHost: 2, IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 10 * time.Second},
		CheckRedirect: func(request *http.Request, previous []*http.Request) error {
			if len(previous) >= 5 || request.URL.Scheme != previous[0].URL.Scheme || request.URL.Host != previous[0].URL.Host {
				return fmt.Errorf("cross-origin or excessive provider redirect denied")
			}
			return nil
		},
	}}, nil
}

func (client *Client) url(native string) string {
	base := client.connection.Endpoint
	if client.connection.Kind == modelconn.LMStudio {
		base = strings.TrimSuffix(base, "/api/v1")
	}
	if client.connection.Kind == modelconn.Ollama {
		return strings.TrimSuffix(base, "/api") + "/api/" + native
	}
	return strings.TrimSuffix(base, "/v1") + "/v1/" + native
}

func (client *Client) do(ctx context.Context, method, endpoint string, body io.Reader) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	if client.key != "" {
		request.Header.Set("Authorization", "Bearer "+client.key)
	}
	response, err := client.http.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("Model server unavailable: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, errors.Join(fmt.Errorf("Model server returned HTTP %d", response.StatusCode), response.Body.Close())
	}
	return response, nil
}

func (client *Client) ListModels(ctx context.Context) (models []modelconn.ModelInfo, err error) {
	endpoint := client.url("models")
	if client.connection.Kind == modelconn.Ollama {
		endpoint = client.url("tags")
	}
	if client.connection.Kind == modelconn.LMStudio && !strings.HasSuffix(client.connection.Endpoint, "/v1") {
		endpoint = strings.TrimSuffix(client.connection.Endpoint, "/api/v1") + "/api/v1/models"
	}
	response, err := client.do(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, response.Body.Close()) }()
	content, err := io.ReadAll(io.LimitReader(response.Body, 4*1024*1024+1))
	if err != nil {
		return nil, err
	}
	if len(content) > 4*1024*1024 {
		return nil, fmt.Errorf("model list exceeds size limit")
	}
	var listing struct {
		Models []struct {
			Name          string `json:"name"`
			Key           string `json:"key"`
			Type          string `json:"type"`
			ContextWindow int    `json:"max_context_length"`
		} `json:"models"`
		Data []struct {
			ID            string `json:"id"`
			ContextWindow int    `json:"max_model_len"`
		} `json:"data"`
	}
	if err := json.Unmarshal(content, &listing); err != nil {
		return nil, fmt.Errorf("invalid model list: %w", err)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(content, &envelope); err != nil {
		return nil, err
	}
	if _, native := envelope["models"]; !native {
		if _, compatible := envelope["data"]; !compatible {
			return nil, fmt.Errorf("unsupported model list format; enter Model ID manually")
		}
	}
	seen := make(map[string]bool)
	appendModel := func(id string, contextWindow int) error {
		if id == "" || len(id) > 512 || strings.ContainsAny(id, "\x00\r\n\x1b") {
			return fmt.Errorf("invalid Model ID")
		}
		if !seen[id] {
			models = append(models, modelconn.ModelInfo{ID: id, ContextWindow: contextWindow})
			seen[id] = true
		}
		if len(models) > 10000 {
			return fmt.Errorf("too many models")
		}
		return nil
	}
	for _, model := range listing.Models {
		if model.Type != "" && model.Type != "llm" {
			continue
		}
		id := model.Name
		if model.Key != "" {
			id = model.Key
		}
		if err := appendModel(id, model.ContextWindow); err != nil {
			return nil, err
		}
	}
	for _, model := range listing.Data {
		if err := appendModel(model.ID, model.ContextWindow); err != nil {
			return nil, err
		}
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
	return models, nil
}

// Close releases this adapter's idle HTTP connections after its task owner has joined.
func (client *Client) Close() { client.http.CloseIdleConnections() }
