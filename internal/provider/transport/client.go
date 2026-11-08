// Purpose: Own authenticated HTTP connections and their shutdown.
// Pattern: HTTP lifecycle adapter.
// Usage: Provider adapters create a Client and close it after all requests finish.
// Related: internal/provider/transport/transport.go.
package transport

import (
	"net"
	"net/http"
	"strings"
	"time"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

type Client struct {
	key      string
	HTTP     *http.Client
	provider connection.ProviderKind
}

func New(key string, provider connection.ProviderKind) *Client {
	return &Client{key: strings.Trim(key, " \t"), provider: provider, HTTP: &http.Client{
		Transport: &http.Transport{
			Proxy:             http.ProxyFromEnvironment,
			DialContext:       (&net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			ForceAttemptHTTP2: true,
			// Retain a small parallel round across HTTP/1.1 fallbacks. This
			// bounds idle sockets only; request admission remains caller-owned.
			MaxIdleConns:        8,
			MaxIdleConnsPerHost: 8,
			IdleConnTimeout:     30 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		},
		CheckRedirect: func(request *http.Request, previous []*http.Request) error {
			if len(previous) >= 5 || request.URL.Scheme != previous[0].URL.Scheme || request.URL.Host != previous[0].URL.Host {
				return faults.Errorf(diagcatalog.TransportCrossOriginOrExcessiveProviderRedirectDenied)
			}
			return nil
		},
	}}
}

func (c *Client) Close() { c.HTTP.CloseIdleConnections() }
