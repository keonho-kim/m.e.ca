// Purpose: Discover published browser dependencies without provisioning or starting them.
// Pattern: Read-only runtime inventory.
// Usage: The browser session consults Available while constructing the request tool catalog.
// Related: internal/tools/browser/runtime/install.go, internal/tools/browser/runtime/managed.go.
package runtime

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func runtimeDirectory(state string) string {
	return filepath.Join(state, "browser", "runtime")
}

// Available checks the installation metadata and executable paths. Opening a
// session still verifies the helper checksum and validates the actual launch.
func Available(state string) bool {
	directory := runtimeDirectory(state)
	if info, err := os.Lstat(directory); err != nil || !info.IsDir() {
		return false
	}
	helper, err := hostAgent()
	if err != nil {
		return false
	}
	path, err := installedPath(directory, helper)
	if err != nil || path == "" {
		return false
	}
	browser, err := findBrowser()
	return err == nil && (browser != "" || availableManagedChrome(directory, runtime.GOOS, runtime.GOARCH))
}

func availableManagedChrome(directory, goos, arch string) bool {
	directory = filepath.Join(directory, "chrome")
	if info, err := os.Lstat(directory); err != nil || !info.IsDir() {
		return false
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return false
	}
	defer root.Close()
	selected, err := readSelection(root)
	if err != nil || selected.Version == "" {
		return false
	}
	value, err := chromeAsset(goos, arch, selected.Version)
	if err != nil {
		return false
	}
	path, err := installedPath(filepath.Join(root.Name(), "versions"), value)
	return err == nil && path != ""
}

func installedPath(parent string, value asset) (string, error) {
	directory := filepath.Join(parent, value.name)
	info, err := os.Lstat(directory)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fault.Errorf(diag.BrowserIntegrity)
	}
	path := filepath.Join(directory, value.executable)
	marker, expected := "verified.sha256", value.digest
	if value.zip {
		marker, expected = "installed.version", value.version
	}
	stamp, err := readRuntimeMarker(filepath.Join(directory, marker))
	if err != nil || stamp != expected || !executable(path) {
		return "", fault.Errorf(diag.BrowserIntegrity)
	}
	return path, nil
}
