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
	"syscall"
	"time"

	"github.com/keonho-kim/meca/internal/config/statefs"
	"github.com/keonho-kim/meca/internal/diagnostics"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type chromeSelection struct {
	Version  string
	Previous string `json:",omitempty"`
	Checked  time.Time
}

func (s chromeSelection) fresh(now time.Time) bool {
	return s.Version != "" && !now.Before(s.Checked) && now.Sub(s.Checked) < 24*time.Hour
}

func managedChrome(ctx context.Context, client *http.Client, directory, goos, arch string, now time.Time) (use chromeUse, err error) {
	root, err := statefs.Open(filepath.Join(directory, "chrome"))
	if err != nil {
		return use, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, use.close())
			use = chromeUse{}
		}
	}()
	defer func() { err = errors.Join(err, root.Close()) }()
	selected, err := readSelection(root)
	if err != nil {
		return use, err
	}
	if selected.fresh(now) {
		return leaseSelected(ctx, root, goos, arch, nil)
	}
	// The update lock spans network and extraction. Readers acquire a separate
	// short lock for the selected version while an update is still in progress.
	updates, err := statefs.Open(filepath.Join(root.Name(), "updates"))
	if err != nil {
		return use, err
	}
	defer func() { err = errors.Join(err, updates.Close()) }()
	update, err := statefs.Lease(updates, true)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		if selected.Version != "" {
			return leaseSelected(ctx, root, goos, arch, nil)
		}
		update, err = statefs.Lock(ctx, updates)
	}
	if err != nil {
		return use, err
	}
	defer func() { err = errors.Join(err, update.Close()) }()
	selected, err = readSelection(root)
	if err != nil {
		return use, err
	}
	if selected.fresh(now) {
		return leaseSelected(ctx, root, goos, arch, nil)
	}
	versions := filepath.Join(root.Name(), "versions")
	if selected.Version != "" {
		if _, _, err := selectedChrome(ctx, versions, goos, arch, selected); err != nil {
			return use, err
		}
	}
	next, updateErr := latestStable(ctx, client, goos, arch)
	if updateErr == nil {
		_, updateErr = install(ctx, client, versions, next)
	}
	if updateErr != nil {
		if ctx.Err() != nil {
			return use, ctx.Err()
		}
		if selected.Version == "" {
			return use, updateErr
		}
		// Preserve the working version and avoid repeated downloads on each request.
	} else if selected.Version != next.version {
		selected.Previous, selected.Version = selected.Version, next.version
	}
	selected.Checked = now
	use, err = leaseSelected(ctx, root, goos, arch, &selected)
	if err != nil {
		return use, err
	}
	if updateErr == nil {
		if err := pruneChrome(ctx, root, goos, arch); err != nil && ctx.Err() == nil {
			diagnostics.Log(ctx, diag.BrowserRuntimeCleanupFailed, "error", err)
		}
	}
	return use, nil
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
	if json.Unmarshal(body, &selected) != nil || !validVersion(selected.Version) || selected.Checked.IsZero() ||
		selected.Previous != "" && (!validVersion(selected.Previous) || selected.Previous == selected.Version) {
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
