// Purpose: Stream fixed runtime downloads to disk with cancellation, size and digest checks.
// Pattern: HTTP acquisition boundary without whole-archive allocation.
// Usage: Installer downloads into its unpublished staging directory.
// Related: internal/tools/browser/runtime/install.go, internal/tools/browser/runtime/assets.go.
package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const maxDownload = 256 << 20

var downloadClient = &http.Client{Timeout: 10 * time.Minute, CheckRedirect: runtimeRedirect}

func runtimeRedirect(req *http.Request, via []*http.Request) error {
	invalid := len(via) > 10 || req.URL.Scheme != "https" || req.URL.User != nil
	if len(via) > 0 {
		switch via[0].URL.Host {
		case "storage.googleapis.com":
			invalid = invalid || req.URL.String() != via[0].URL.String()
		case "googlechromelabs.github.io":
			invalid = invalid || req.URL.String() != via[0].URL.String()
		}
	}
	if invalid {
		return fault.Errorf(diag.BrowserInput, diag.Text(diag.BrowserInvalidRedirect))
	}
	return nil
}

func download(ctx context.Context, client *http.Client, value asset, path string) (err error) {
	if value.zip && !strings.HasPrefix(value.url, "https://storage.googleapis.com/chrome-for-testing-public/") {
		return fault.Errorf(diag.BrowserStableMetadata)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, value.url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, response.Body.Close()) }()
	if response.StatusCode != http.StatusOK {
		return fault.Errorf(diag.BrowserDownload, response.StatusCode)
	}
	if response.ContentLength > maxDownload {
		return fault.Errorf(diag.BrowserLimit)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	digest := sha256.New()
	var writer io.Writer = file
	if value.digest != "" {
		writer = io.MultiWriter(file, digest)
	}
	n, err := io.Copy(writer, io.LimitReader(response.Body, maxDownload+1))
	if err != nil {
		return err
	}
	if n > maxDownload {
		return fault.Errorf(diag.BrowserLimit)
	}
	if value.digest != "" && hex.EncodeToString(digest.Sum(nil)) != value.digest {
		return fault.Errorf(diag.BrowserIntegrity)
	}
	return file.Sync()
}
