// Purpose: Prepare a verified helper and browser once under a process-shared installation lease.
// Pattern: Lazy runtime provisioning with atomic directory publication.
// Usage: Browser sessions call Ensure at first open; MECA startup never downloads anything.
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

type Paths struct{ Agent, Browser string }

func Ensure(ctx context.Context, state string) (Paths, error) {
	directory := filepath.Join(state, "browser", "runtime")
	helper, err := hostAgent()
	if err != nil {
		return Paths{}, err
	}
	browser, err := findBrowser()
	if err != nil {
		return Paths{}, err
	}
	agent, err := install(ctx, downloadClient, directory, helper)
	if err != nil {
		return Paths{}, fault.Errorf(diag.BrowserRuntime, err)
	}
	if browser == "" {
		browser, err = managedChrome(ctx, downloadClient, directory, runtime.GOOS, runtime.GOARCH, time.Now())
		if err != nil {
			return Paths{}, fault.Errorf(diag.BrowserRuntime, err)
		}
	}
	return Paths{agent, browser}, nil
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
	lease, err := statefs.Lock(ctx, root)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()

	destination := filepath.Join(parent, value.name)
	path = filepath.Join(destination, value.executable)
	if info, statErr := os.Lstat(destination); statErr == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fault.Errorf(diag.BrowserIntegrity)
		}
		marker, expected := "verified.sha256", value.digest
		if value.zip {
			marker, expected = "installed.version", value.version
		}
		stamp, readErr := os.ReadFile(filepath.Join(destination, marker))
		if readErr != nil || string(stamp) != expected || !executable(path) {
			return "", fault.Errorf(diag.BrowserIntegrity)
		}
		if !value.zip {
			file, err := os.Open(path)
			if err != nil {
				return "", err
			}
			hash := sha256.New()
			_, readErr := io.Copy(hash, file)
			closeErr := file.Close()
			if readErr != nil || closeErr != nil {
				return "", errors.Join(readErr, closeErr)
			}
			if hex.EncodeToString(hash.Sum(nil)) != value.digest {
				return "", fault.Errorf(diag.BrowserIntegrity)
			}
		}
		return path, nil
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return "", statErr
	}
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
