// Purpose: Protect a managed Chrome version until its owning session confirms shutdown.
// Pattern: On-disk use marker and short selection lock shared by all MECA processes.
// Usage: Runtime preparation pins the selected version and Session.Close removes the pin.
// Related: internal/tools/browser/runtime/managed.go, internal/tools/browser/session.go.
package runtime

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/keonho-kim/meca/internal/config/statefs"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const leaseAwareMarker = ".meca-use-markers"
const chromeUsePrefix = ".meca-active-"
const retiredChromePrefix = ".meca-retired-"

type chromeUse struct {
	path, marker string
}

func (use chromeUse) close() error {
	if use.marker == "" {
		return nil
	}
	err := os.Remove(use.marker)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func selectedChrome(ctx context.Context, versions, goos, arch string, selected chromeSelection) (asset, string, error) {
	value, err := chromeAsset(goos, arch, selected.Version)
	if err != nil {
		return asset{}, "", err
	}
	path, err := installed(ctx, versions, value)
	if err == nil && path == "" {
		err = fault.Errorf(diag.BrowserIntegrity)
	}
	return value, path, err
}

func leaseSelected(ctx context.Context, root *os.Root, goos, arch string, next *chromeSelection) (use chromeUse, err error) {
	lease, err := statefs.Lock(ctx, root)
	if err != nil {
		return use, err
	}
	defer func() {
		err = errors.Join(err, lease.Close())
		if err != nil {
			err = errors.Join(err, use.close())
			use = chromeUse{}
		}
	}()
	selected, err := readSelection(root)
	if err != nil {
		return use, err
	}
	if next != nil {
		selected = *next
	}
	versions := filepath.Join(root.Name(), "versions")
	value, path, err := selectedChrome(ctx, versions, goos, arch, selected)
	if err != nil {
		return use, err
	}
	marker := filepath.Join(versions, value.name, chromeUsePrefix+rand.Text())
	file, err := os.OpenFile(marker, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return use, err
	}
	use = chromeUse{path: path, marker: marker}
	if err := file.Close(); err != nil {
		return use, err
	}
	if next != nil {
		body, err := json.Marshal(selected)
		if err != nil {
			return use, err
		}
		if err := statefs.Replace(root, "current.json", body); err != nil {
			return use, err
		}
	}
	return use, nil
}

func pruneChrome(ctx context.Context, root *os.Root, goos, arch string) error {
	lease, err := statefs.Lock(ctx, root)
	if err != nil {
		return err
	}
	keep, err := readSelection(root)
	if err != nil {
		return errors.Join(err, lease.Close())
	}
	versions := filepath.Join(root.Name(), "versions")
	retired, detachErr := detachUnusedChrome(versions, goos, arch, keep)
	err = errors.Join(detachErr, lease.Close())
	// Removing an archive may take time. Readers can select their version
	// once unleased versions have been detached from the published directory.
	for _, directory := range retired {
		err = errors.Join(err, os.RemoveAll(directory))
	}
	return err
}

func detachUnusedChrome(versions, goos, arch string, keep chromeSelection) ([]string, error) {
	entries, err := os.ReadDir(versions)
	if err != nil {
		return nil, err
	}
	var retired []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(versions, entry.Name())
		suffix, retiredName := strings.CutPrefix(entry.Name(), retiredChromePrefix)
		if retiredName && len(suffix) == 26 && strings.Trim(suffix, "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567") == "" {
			retired = append(retired, path)
			continue
		}
		if !strings.HasPrefix(entry.Name(), "chrome-") {
			continue
		}
		version, err := readRuntimeMarker(filepath.Join(path, leaseAwareMarker))
		if errors.Is(err, os.ErrNotExist) {
			continue // Ownership of installations that predate use markers is unknown.
		}
		if err != nil {
			return retired, err
		}
		value, err := chromeAsset(goos, arch, version)
		if err != nil || value.name != entry.Name() || version == keep.Version || version == keep.Previous {
			continue
		}
		files, err := os.ReadDir(path)
		if err != nil {
			return retired, err
		}
		active := false
		for _, file := range files {
			if strings.HasPrefix(file.Name(), chromeUsePrefix) {
				active = true
				break
			}
		}
		if active {
			continue
		}
		destination := filepath.Join(versions, retiredChromePrefix+rand.Text())
		if err := os.Rename(path, destination); err != nil {
			return retired, err
		}
		retired = append(retired, destination)
	}
	return retired, nil
}
