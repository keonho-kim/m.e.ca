// Purpose: Own validated provider connections and model discovery result values.
// Pattern: Simple Module (model-selection boundary).
// Usage: Used by provider construction, model menus and persisted connections.
// Related: internal/provider/discovery/client.go, internal/storage/sqlite/selection.go, SYSTEM.md.

package connection

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"strings"
)

type Connection struct {
	ID            string       `json:"id"`
	Kind          ProviderKind `json:"kind"`
	Endpoint      string       `json:"endpoint"`
	CredentialRef string       `json:"credential_ref"`
}

func New(kind ProviderKind, endpoint string) (Connection, error) {
	defaultEndpoint := ""
	for _, provider := range Providers() {
		if provider.Kind == kind {
			defaultEndpoint = provider.Endpoint
			break
		}
	}
	if defaultEndpoint == "" {
		return Connection{}, fmt.Errorf("unknown provider: %s", kind)
	}
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	if len(endpoint) > 4096 {
		return Connection{}, fmt.Errorf("endpoint is too long")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Hostname() == "" || parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return Connection{}, fmt.Errorf("endpoint must be an HTTP(S) URL without credentials, query or fragment")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	endpoint = parsed.String()
	id := sha256.Sum256([]byte(string(kind) + "\n" + endpoint))
	return Connection{ID: fmt.Sprintf("provider_%x", id[:12]), Kind: kind, Endpoint: endpoint}, nil
}
