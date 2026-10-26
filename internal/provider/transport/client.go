// Purpose: Own authenticated HTTP connections and their shutdown.
// Pattern: HTTP lifecycle adapter.
// Usage: Provider adapters create a Client and close it after all requests finish.
// Related: internal/provider/transport/transport.go.
package transport

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

type Client struct {
	key  string
	HTTP *http.Client
}

func New(key string) *Client {
	return &Client{key: key, HTTP: &http.Client{
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			DialContext:         (&net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			ForceAttemptHTTP2:   true,
			MaxIdleConns:        2,
			MaxIdleConnsPerHost: 2,
			IdleConnTimeout:     30 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		},
		CheckRedirect: func(request *http.Request, previous []*http.Request) error {
			if len(previous) >= 5 || request.URL.Scheme != previous[0].URL.Scheme || request.URL.Host != previous[0].URL.Host {
				return fmt.Errorf("cross-origin or excessive provider redirect denied")
			}
			return nil
		},
	}}
}

func (c *Client) Close() { c.HTTP.CloseIdleConnections() }
