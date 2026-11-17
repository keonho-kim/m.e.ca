// Purpose: Extract a verified Chrome ZIP within a new directory using bounded writes and contained links.
// Pattern: Archive filesystem trust boundary.
// Usage: Installer publishes the staging directory only after every entry succeeds.
// Related: internal/tools/browser/runtime/install.go.
package runtime

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func extract(ctx context.Context, archive, directory string) (err error) {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, reader.Close()) }()
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	if len(reader.File) > 50000 {
		return fault.Errorf(diag.BrowserArchive)
	}
	var total uint64
	type link struct{ name, target string }
	var links []link
	for _, entry := range reader.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := strings.TrimSuffix(entry.Name, "/")
		if !filepath.IsLocal(name) || strings.Contains(name, "\\") {
			return fault.Errorf(diag.BrowserArchive)
		}
		mode := entry.Mode()
		if err := root.MkdirAll(filepath.Dir(name), 0700); err != nil {
			return err
		}
		if mode.IsDir() {
			if err := root.MkdirAll(name, 0700); err != nil {
				return err
			}
			continue
		}
		if entry.UncompressedSize64 > (1<<30)-total || entry.UncompressedSize64 > 512<<20 {
			return fault.Errorf(diag.BrowserArchive)
		}
		total += entry.UncompressedSize64
		source, err := entry.Open()
		if err != nil {
			return err
		}
		if mode&os.ModeSymlink != 0 {
			target, readErr := io.ReadAll(io.LimitReader(source, 4097))
			closeErr := source.Close()
			if readErr != nil || closeErr != nil {
				return errors.Join(readErr, closeErr)
			}
			resolved := filepath.Clean(filepath.Join(filepath.Dir(name), string(target)))
			if len(target) > 4096 || filepath.IsAbs(string(target)) || !filepath.IsLocal(resolved) {
				return fault.Errorf(diag.BrowserArchive)
			}
			links = append(links, link{name, string(target)})
			continue
		}
		if !mode.IsRegular() {
			source.Close()
			return fault.Errorf(diag.BrowserArchive)
		}
		permission := os.FileMode(0600)
		if mode.Perm()&0111 != 0 {
			permission = 0700
		}
		target, openErr := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, permission)
		if openErr != nil {
			source.Close()
			return openErr
		}
		_, copyErr := io.Copy(target, contextReader{ctx, io.LimitReader(source, int64(entry.UncompressedSize64)+1)})
		if err := errors.Join(copyErr, target.Close(), source.Close()); err != nil {
			return err
		}
	}
	// Chrome frameworks contain relative links. Create them last so archive writes
	// can never traverse an earlier archive-provided symlink.
	for _, item := range links {
		if err := root.Symlink(item.target, item.name); err != nil {
			return err
		}
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	source io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.source.Read(p)
}
