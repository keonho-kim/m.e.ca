// Purpose: Read GitHub release responses with bounded retries and origin-scoped credentials.
// Pattern: HTTP boundary adapter.
// Usage: NewClient owns its transport; release discovery and asset downloads call read.
// Related: internal/commands/upgrade/release.go, internal/commands/upgrade/credential.go.
package upgrade

import (
	"context"
	"errors"

	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const maxInfoBytes = 16 * 1024 * 1024

var errTooLarge = faults.Errorf(diagcatalog.UpgradeReleaseResponseExceedsItsSizeLimit)

type Client struct {
	http    *http.Client
	baseURL string
	token   string
}

func NewClient(token string) (*Client, error) {
	token = strings.TrimSpace(token)
	for _, character := range token {
		if character <= ' ' || character > '~' {
			return nil, faults.Errorf(diagcatalog.UpgradeGitHubCredentialMustBeOneASCIIToken)
		}
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 30 * time.Second
	return &Client{
		http:    &http.Client{Transport: transport, CheckRedirect: releaseRedirect},
		baseURL: "https://api.github.com/repos/keonho-kim/m.e.ca", token: token,
	}, nil
}

func (c *Client) Close() { c.http.CloseIdleConnections() }

func releaseRedirect(request *http.Request, previous []*http.Request) error {
	if len(previous) >= 10 {
		return faults.Errorf(diagcatalog.UpgradeTooManyReleaseDownloadRedirects)
	}
	if request.URL.Scheme != "https" {
		return faults.Errorf(diagcatalog.UpgradeReleaseDownloadRequiresHTTPS)
	}
	if request.URL.Host != previous[0].URL.Host {
		request.Header.Del("Authorization")
	}
	return nil
}

func (c *Client) read(ctx context.Context, path, accept string, consume func(io.Reader) error) error {
	for attempt := 0; attempt < 3; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
		if err != nil {
			return faults.Errorf(diagcatalog.UpgradeInvalidGitHubReleaseEndpoint)
		}
		request.Header.Set("Accept", accept)
		request.Header.Set("X-GitHub-Api-Version", "2026-03-10")
		request.Header.Set("User-Agent", "meca-upgrade")
		if c.token != "" {
			request.Header.Set("Authorization", "Bearer "+c.token)
		}
		response, err := c.http.Do(request)
		retry := true
		if err == nil {
			if response.StatusCode == http.StatusOK {
				err = consume(response.Body)
			} else {
				err = faults.Errorf(diagcatalog.UpgradeGitHubReturnedHTTP, response.StatusCode)
				if response.StatusCode == 401 || response.StatusCode == 403 || response.StatusCode == 404 {
					err = faults.Errorf(diagcatalog.UpgradeReleaseAccessFailedHTTPCheckRepositoryAccess, response.StatusCode)
				}
				retry = response.StatusCode == 429 || response.StatusCode >= 500
			}
			err = errors.Join(err, response.Body.Close())
		}
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var pathError *os.PathError
		if !retry || errors.Is(err, errTooLarge) || errors.As(err, &pathError) || attempt == 2 {
			var requestError *url.Error
			if errors.As(err, &requestError) {
				err = requestError.Err // Signed asset URLs must not appear in errors.
			}
			return faults.Errorf(diagcatalog.UpgradeGitHubReleaseRequestFailedAfterAttemptS, attempt+1, err)
		}
		timer := time.NewTimer(time.Duration(250*(1<<attempt)) * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return nil
}

func readInfo(reader io.Reader) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, maxInfoBytes+1))
	if len(body) > maxInfoBytes {
		return nil, errTooLarge
	}
	return body, err
}
