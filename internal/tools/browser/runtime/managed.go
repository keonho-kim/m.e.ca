// Purpose: Select a verified managed Chrome version without replacing running sessions' executable paths.
// Pattern: Process-shared installation transaction with a bounded current-version record.
// Usage: Ensure uses the managed browser when no system browser was selected.
// Related: internal/tools/browser/runtime/stable.go, internal/tools/browser/runtime/install.go.
package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/keonho-kim/meca/internal/config/statefs"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type chromeSelection struct {
	Version string
	Checked time.Time
}

func managedChrome(ctx context.Context, client *http.Client, directory, goos, arch string, now time.Time) (path string, err error) {
	root, err := statefs.Open(filepath.Join(directory, "chrome"))
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	lease, err := statefs.Lock(ctx, root)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()
	selected, err := readSelection(root)
	if err != nil {
		return "", err
	}
	versions := filepath.Join(root.Name(), "versions")
	if selected.Version != "" {
		value, e := chromeAsset(goos, arch, selected.Version)
		if e != nil {
			return "", e
		}
		// Cache reads never redownload or repair missing/corrupted installations.
		path = filepath.Join(versions, value.name, value.executable)
		info, e := os.Lstat(filepath.Join(versions, value.name))
		if e != nil {
			return "", e
		}
		stamp, e := os.ReadFile(filepath.Join(versions, value.name, "installed.version"))
		if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || string(stamp) != value.version || !executable(path) {
			return "", fault.Errorf(diag.BrowserIntegrity)
		}
		if !now.Before(selected.Checked) && now.Sub(selected.Checked) < 24*time.Hour {
			return path, nil
		}
	}
	next, updateErr := latestStable(ctx, client, goos, arch)
	if updateErr == nil {
		_, updateErr = install(ctx, client, versions, next)
	}
	if updateErr != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if path == "" {
			return "", updateErr
		}
		// Preserve the working version and avoid repeated downloads on each request.
		selected.Checked = now
	} else {
		selected = chromeSelection{Version: next.version, Checked: now}
		path = filepath.Join(versions, next.name, next.executable)
	}
	body, err := json.Marshal(selected)
	if err != nil {
		return "", err
	}
	if err := statefs.Write(root.Name(), "current.json", body); err != nil {
		return "", err
	}
	return path, nil
}

func readSelection(root *os.Root) (chromeSelection, error) {
	var selected chromeSelection
	file, err := root.Open("current.json")
	if errors.Is(err, os.ErrNotExist) {
		return selected, nil
	}
	if err != nil {
		return selected, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return selected, err
	}
	if !info.Mode().IsRegular() || info.Size() > 4096 {
		return selected, fault.Errorf(diag.BrowserStableMetadata)
	}
	body, err := io.ReadAll(io.LimitReader(file, 4097))
	if err != nil {
		return selected, err
	}
	if json.Unmarshal(body, &selected) != nil || !validVersion(selected.Version) || selected.Checked.IsZero() {
		return selected, fault.Errorf(diag.BrowserStableMetadata)
	}
	return selected, nil
}

func verifyChrome(ctx context.Context, path, version string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, path, "--version")
	command.WaitDelay = 2 * time.Second
	output := &versionOutput{}
	command.Stdout, command.Stderr = output, output
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if dependency := MissingLibrary(output.body); dependency != nil {
			return dependency
		}
		return err
	}
	if output.exceeded {
		return fault.Errorf(diag.BrowserStableVerification)
	}
	for _, field := range strings.Fields(string(output.body)) {
		if field == version {
			return nil
		}
	}
	return fault.Errorf(diag.BrowserStableVerification)
}

type versionOutput struct {
	body     []byte
	exceeded bool
}

func (v *versionOutput) Write(p []byte) (int, error) {
	if len(p) > 4096-len(v.body) {
		v.exceeded = true
		return 0, fault.Errorf(diag.BrowserLimit)
	}
	v.body = append(v.body, p...)
	return len(p), nil
}
