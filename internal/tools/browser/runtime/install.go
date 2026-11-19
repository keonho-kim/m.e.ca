// Purpose: Verify installed browser dependencies and provision missing ones on explicit setup.
// Pattern: Runtime provisioning with atomic directory publication.
// Usage: Tool sessions open installed dependencies; the extension installer may provision them.
// Related: internal/tools/browser/runtime/download.go, internal/config/statefs/lock.go.
package runtime

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/keonho-kim/meca/internal/config/statefs"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

//go:embed agent-browser.LICENSE
var helperLicense []byte

type Paths struct {
	Agent, Browser string
	chrome         chromeUse
}

// Close releases the managed version after the owning browser confirms shutdown.
func (p Paths) Close() error { return p.chrome.close() }

func OpenInstalled(ctx context.Context, state string) (Paths, error) {
	return prepareRuntime(ctx, state, false)
}

func Install(ctx context.Context, state string) (Paths, error) {
	return prepareRuntime(ctx, state, true)
}

func prepareRuntime(ctx context.Context, state string, installMissing bool) (Paths, error) {
	directory := runtimeDirectory(state)
	helper, err := hostAgent()
	if err != nil {
		return Paths{}, err
	}
	browser, err := findBrowser()
	if err != nil {
		return Paths{}, err
	}
	var agent string
	if installMissing {
		agent, err = install(ctx, downloadClient, directory, helper)
	} else {
		agent, err = openInstalledAgent(ctx, directory, helper)
	}
	if err != nil {
		return Paths{}, preparationFailure(err)
	}
	if agent == "" {
		return Paths{}, fault.New(diag.FailedPrecondition, diag.BrowserRuntimeMissing)
	}
	paths := Paths{Agent: agent, Browser: browser}
	if browser == "" {
		paths.chrome, err = managedChrome(ctx, downloadClient, directory, runtime.GOOS, runtime.GOARCH, time.Now(), installMissing)
		if err != nil {
			return Paths{}, preparationFailure(err)
		}
		paths.Browser = paths.chrome.path
	}
	return paths, nil
}

func openInstalledAgent(ctx context.Context, directory string, helper asset) (string, error) {
	if _, err := os.Lstat(directory); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	root, err := statefs.Open(directory)
	if err != nil {
		return "", err
	}
	path, err := installed(ctx, root.Name(), helper)
	return path, errors.Join(err, root.Close())
}

// Preserve actionable catalog messages, including loader diagnostics, at the CLI boundary.
func preparationFailure(err error) error {
	var known *fault.Error
	if errors.As(err, &known) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return fault.Errorf(diag.BrowserRuntime, err)
}

func findBrowser() (string, error) {
	if path := os.Getenv("MECA_BROWSER_EXECUTABLE"); path != "" {
		if !filepath.IsAbs(path) || !executable(path) {
			return "", fault.Errorf(diag.BrowserInput, diag.Text(diag.BrowserInvalidExecutable))
		}
		return filepath.EvalSymlinks(path)
	}
	candidates := []string{"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "/Applications/Chromium.app/Contents/MacOS/Chromium"}
	if runtime.GOOS == "linux" {
		candidates = []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable"}
	}
	for _, path := range candidates {
		if !filepath.IsAbs(path) {
			resolved, err := exec.LookPath(path)
			if err != nil {
				continue
			}
			path = resolved
		}
		if executable(path) {
			return filepath.EvalSymlinks(path)
		}
	}
	return "", nil
}

func install(ctx context.Context, client *http.Client, parent string, value asset) (path string, err error) {
	root, err := statefs.Open(parent)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	if path, err := installed(ctx, parent, value); path != "" || err != nil {
		return path, err
	}
	lease, err := statefs.Lock(ctx, root)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()
	if path, err := installed(ctx, parent, value); path != "" || err != nil {
		return path, err
	}

	destination := filepath.Join(parent, value.name)
	path = filepath.Join(destination, value.executable)
	stage, err := os.MkdirTemp(parent, ".preparing-")
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(stage)) }()
	downloaded := filepath.Join(stage, "download")
	if err := download(ctx, client, value, downloaded); err != nil {
		return "", err
	}
	if value.zip {
		if err := extract(ctx, downloaded, stage); err != nil {
			return "", err
		}
		if err := os.Remove(downloaded); err != nil {
			return "", err
		}
	} else {
		if err := os.Rename(downloaded, filepath.Join(stage, value.executable)); err != nil {
			return "", err
		}
		if err := os.Chmod(filepath.Join(stage, value.executable), 0700); err != nil {
			return "", err
		}
	}
	if !executable(filepath.Join(stage, value.executable)) {
		return "", fault.Errorf(diag.BrowserIntegrity)
	}
	if !value.zip {
		if err := os.WriteFile(filepath.Join(stage, "LICENSE"), helperLicense, 0600); err != nil {
			return "", err
		}
	}
	marker, stamp := "verified.sha256", value.digest
	if value.zip {
		if err := verifyChrome(ctx, filepath.Join(stage, value.executable), value.version); err != nil {
			return "", err
		}
		marker, stamp = "installed.version", value.version
		if err := os.WriteFile(filepath.Join(stage, leaseAwareMarker), []byte(value.version), 0600); err != nil {
			return "", err
		}
	}
	if err := os.WriteFile(filepath.Join(stage, marker), []byte(stamp), 0600); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := os.Rename(stage, destination); err != nil {
		return "", err
	}
	return path, nil
}

// Published assets are immutable to the installer. Warm reads can validate
// concurrently while still checking the entire helper against its pinned hash.
func installed(ctx context.Context, parent string, value asset) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path, err := installedPath(parent, value)
	if err != nil || path == "" {
		return path, err
	}
	if !value.zip {
		file, err := os.Open(path)
		if err != nil {
			return "", err
		}
		hash := sha256.New()
		_, readErr := io.Copy(hash, file)
		if err := errors.Join(readErr, file.Close()); err != nil {
			return "", err
		}
		if hex.EncodeToString(hash.Sum(nil)) != value.digest {
			return "", fault.Errorf(diag.BrowserIntegrity)
		}
	}
	return path, ctx.Err()
}

func readRuntimeMarker(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() > 64 {
		return "", fault.Errorf(diag.BrowserIntegrity)
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	body, readErr := io.ReadAll(io.LimitReader(file, 65))
	return string(body), errors.Join(readErr, file.Close())
}
