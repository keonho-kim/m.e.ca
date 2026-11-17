// Purpose: Resolve the official Stable channel into an exact platform-specific Chrome archive.
// Pattern: Bounded HTTPS metadata boundary with strict version and URL validation.
// Usage: Managed Chrome checks this endpoint before initial installation and periodic updates.
// Related: internal/tools/browser/runtime/assets.go, internal/tools/browser/runtime/managed.go.
package runtime

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const stableEndpoint = "https://googlechromelabs.github.io/chrome-for-testing/last-known-good-versions-with-downloads.json"

func validVersion(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		number, err := strconv.ParseUint(part, 10, 32)
		if err != nil || strconv.FormatUint(number, 10) != part {
			return false
		}
	}
	return true
}

func latestStable(ctx context.Context, client *http.Client, goos, arch string) (asset, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, stableEndpoint, nil)
	if err != nil {
		return asset{}, err
	}
	res, err := client.Do(req)
	if err != nil {
		return asset{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return asset{}, fault.Errorf(diag.BrowserDownload, res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, (1<<20)+1))
	if err != nil {
		return asset{}, err
	}
	if len(body) > 1<<20 {
		return asset{}, fault.Errorf(diag.BrowserLimit)
	}
	var metadata struct {
		Channels struct {
			Stable struct {
				Version   string
				Downloads struct {
					Chrome []struct{ Platform, URL string }
				}
			}
		}
	}
	if err := json.Unmarshal(body, &metadata); err != nil {
		return asset{}, fault.Errorf(diag.BrowserStableMetadata)
	}
	value, err := chromeAsset(goos, arch, metadata.Channels.Stable.Version)
	if err != nil {
		return asset{}, err
	}
	for _, item := range metadata.Channels.Stable.Downloads.Chrome {
		if item.URL == value.url && item.Platform == strings.TrimPrefix(value.name, "chrome-"+value.version+"-") {
			return value, nil
		}
	}
	return asset{}, fault.Errorf(diag.BrowserStableMetadata)
}
