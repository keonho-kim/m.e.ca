// Purpose: Read a document over connections pinned to validated destination addresses.
// Pattern: Document-scoped HTTP transport with explicit redirect checks.
// Usage: Each first-page read owns its connection and shares it with same-origin render resources.
// Related: internal/tools/browser/webfetch/destination.go, internal/tools/browser/webfetch/document.go.
package webfetch

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"time"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/tools/browser/chromium"
)

type Client struct {
	lookup func(context.Context, string, string) ([]netip.Addr, error)
	dial   func(context.Context, string, string) (net.Conn, error)
}

func NewClient() *Client {
	dialer := &net.Dialer{KeepAlive: 30 * time.Second}
	return &Client{lookup: net.DefaultResolver.LookupNetIP, dial: dialer.DialContext}
}

type documentConnection struct {
	target    destination
	transport *http.Transport
}

func (c *Client) connect(target destination) *documentConnection {
	authority := net.JoinHostPort(target.host, target.port)
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			if address != authority {
				return nil, fault.New(diag.PermissionDenied, diag.WebFetchBlocked)
			}
			ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
			defer cancel()
			deadline, _ := ctx.Deadline()
			var lastErr error
			for index, ip := range target.ips {
				remaining := len(target.ips) - index
				attempt, done := context.WithTimeout(ctx, time.Until(deadline)/time.Duration(remaining))
				connection, err := c.dial(attempt, network, net.JoinHostPort(ip.String(), target.port))
				done()
				if err == nil {
					return connection, nil
				}
				lastErr = err
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
			}
			return nil, lastErr
		},
		MaxIdleConns:           2,
		MaxIdleConnsPerHost:    2,
		IdleConnTimeout:        30 * time.Second,
		ResponseHeaderTimeout:  8 * time.Second,
		TLSHandshakeTimeout:    5 * time.Second,
		MaxResponseHeaderBytes: 32 << 10,
	}
	configureOfflineTLS(transport)
	return &documentConnection{target: target, transport: transport}
}

type openedDocument struct {
	address    *url.URL
	response   *http.Response
	connection *documentConnection
}

func (c *Client) open(ctx context.Context, target *url.URL, userInput string) (result openedDocument, err error) {
	selected, err := c.resolve(ctx, target)
	if err != nil {
		return result, err
	}
	if selected.scope == privateAddress && !userSuppliedURL(userInput, target) {
		return result, fault.New(diag.PermissionDenied, diag.WebFetchPrivateURL)
	}
	connection := c.connect(selected)
	defer func() {
		if err != nil {
			connection.transport.CloseIdleConnections()
		}
	}()
	for redirects := 0; ; redirects++ {
		response, err := connection.read(ctx, http.MethodGet, target, "text/html, text/plain, text/markdown")
		if err != nil {
			return result, err
		}
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			return openedDocument{address: target, response: response, connection: connection}, nil
		}
		if !redirectStatus(response.StatusCode) || response.Header.Get("Location") == "" {
			return result, errors.Join(fault.Errorf(diag.WebFetchStatus, response.StatusCode), response.Body.Close())
		}
		if redirects >= 4 {
			return result, errors.Join(fault.Errorf(diag.WebFetchRedirect), response.Body.Close())
		}
		next, err := redirectURL(target, response.Header.Get("Location"))
		// Draining small known responses allows reuse on same-origin redirects.
		if response.ContentLength >= 0 && response.ContentLength <= 2048 {
			_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 2048))
		}
		if err := errors.Join(err, response.Body.Close()); err != nil {
			return result, err
		}
		if urlOrigin(next) != connection.target.origin {
			if connection.target.scope == privateAddress {
				return result, fault.New(diag.PermissionDenied, diag.WebFetchBlocked)
			}
			redirected, err := c.resolve(ctx, next)
			if err != nil {
				return result, err
			}
			if redirected.scope != publicAddress {
				return result, fault.New(diag.PermissionDenied, diag.WebFetchBlocked)
			}
			connection.transport.CloseIdleConnections()
			connection = c.connect(redirected)
		}
		target = next
	}
}

func redirectStatus(status int) bool {
	switch status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

func redirectURL(base *url.URL, location string) (*url.URL, error) {
	target, err := base.Parse(location)
	if err != nil {
		return nil, fault.New(diag.InvalidArgument, diag.WebFetchURL)
	}
	return parseURL(target.String())
}

func (c *documentConnection) read(ctx context.Context, method string, target *url.URL, accept string) (*http.Response, error) {
	if urlOrigin(target) != c.target.origin {
		return nil, fault.New(diag.PermissionDenied, diag.WebFetchBlocked)
	}
	req, err := http.NewRequestWithContext(ctx, method, target.String(), nil)
	if err != nil {
		return nil, fault.New(diag.InvalidArgument, diag.WebFetchURL)
	}
	req.Header.Set("User-Agent", "MECA WebFetch")
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	response, err := c.transport.RoundTrip(req)
	if err != nil {
		return nil, fault.Errorf(diag.WebFetchTransfer, err)
	}
	return response, nil
}

func (c *documentConnection) loadResource(ctx context.Context, address, method string, limit int) (resource chromium.Response, err error) {
	if (method != http.MethodGet && method != http.MethodHead) || limit <= 0 {
		return resource, fault.New(diag.PermissionDenied, diag.WebFetchBlocked)
	}
	target, err := parseURL(address)
	if err != nil {
		return resource, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	response, err := c.read(ctx, method, target, "")
	if err != nil {
		return resource, err
	}
	defer func() { err = errors.Join(err, response.Body.Close()) }()
	if response.StatusCode < 200 {
		return resource, fault.Errorf(diag.WebFetchStatus, response.StatusCode)
	}
	if method == http.MethodGet && response.ContentLength > int64(limit) {
		return resource, fault.Errorf(diag.WebFetchBodyLimit)
	}
	// Leave same-origin redirects to Chromium so it can compute the response URL.
	if location := response.Header.Get("Location"); location != "" && redirectStatus(response.StatusCode) {
		next, err := redirectURL(target, location)
		if err != nil || urlOrigin(next) != c.target.origin {
			return resource, fault.New(diag.PermissionDenied, diag.WebFetchBlocked)
		}
	}
	resource = chromium.Response{Status: response.StatusCode, Header: response.Header}
	if method != http.MethodHead {
		resource.Body, err = readBody(response.Body, limit)
	}
	return resource, err
}

func readBody(reader io.Reader, limit int) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, int64(limit)+1))
	if err == nil && len(body) > limit {
		err = fault.New(diag.ResourceExhausted, diag.WebFetchBodyLimit)
	}
	return body, err
}
