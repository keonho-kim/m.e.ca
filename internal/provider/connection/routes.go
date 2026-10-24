// Purpose: Normalize provider API roots without changing stored connection identities.
// Pattern: Simple Module.
// Usage: Provider discovery and generation use Connection.APIBase.
// Related: internal/provider/discovery/client.go, internal/provider/httpclient/request.go.
package connection

import "strings"

func (c Connection) APIBase() string {
	base := c.Endpoint
	if c.Kind == OpenRouter {
		base = strings.TrimSuffix(base, "/v1")
		return strings.TrimSuffix(base, "/api") + "/api/v1"
	}
	if c.Kind == LMStudio {
		base = strings.TrimSuffix(base, "/api/v1")
	}
	if c.Kind == Ollama {
		return strings.TrimSuffix(base, "/api") + "/api"
	}
	return strings.TrimSuffix(base, "/v1") + "/v1"
}
