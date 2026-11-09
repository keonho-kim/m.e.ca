// Purpose: Own one verified resource descriptor through sniffing, paging and optional image loading.
// Pattern: Explicit filesystem read lease.
// Usage: Server ReadFile and local skill tools close ResourceFile after their complete read.
// Related: internal/skills/resources.go, internal/skills/assets.go, internal/tools/local/skill_image.go.
package skills

import (
	"context"
	"errors"
	"github.com/keonho-kim/meca/internal/agent/media"
	"io"
	"os"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type ResourceFile struct {
	reader   *Reader
	root     *os.Root
	file     *os.File
	info     os.FileInfo
	path     string
	ID       string
	Revision string
}

func (r *Reader) OpenFile(ctx context.Context, id, path string) (*ResourceFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root, canonicalID, err := r.resourceRoot(ctx, id, path)
	if err != nil {
		return nil, err
	}
	file, err := openRegular(root, path, MaxResourceFileBytes)
	if err != nil {
		return nil, errors.Join(err, root.Close())
	}
	info, err := file.Stat()
	if err != nil {
		return nil, errors.Join(err, file.Close(), root.Close())
	}
	return &ResourceFile{reader: r, root: root, file: file, info: info, path: path, ID: canonicalID, Revision: resourceRevision(info, r.version())}, nil
}

func (f *ResourceFile) Close() error { return errors.Join(f.file.Close(), f.root.Close()) }

func (f *ResourceFile) check(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	after, err := f.root.Stat(f.path)
	if err != nil {
		return err
	}
	if f.Revision != resourceRevision(after, f.reader.version()) || !os.SameFile(f.info, after) || f.info.Size() != after.Size() || !f.info.ModTime().Equal(after.ModTime()) {
		return faults.Errorf(diagcatalog.SkillsReferenceChangedDuringRead, ErrInvalid)
	}
	return nil
}

func (f *ResourceFile) ReadBytes(ctx context.Context, maximum int64) ([]byte, error) {
	if maximum < 1 || maximum > media.MaxBytes || f.info.Size() > maximum {
		return nil, ErrLimit
	}
	if err := f.check(ctx); err != nil {
		return nil, err
	}
	body := make([]byte, f.info.Size())
	for offset := 0; offset < len(body); {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		end := min(len(body), offset+(64<<10))
		n, err := f.file.ReadAt(body[offset:end], int64(offset))
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		if n != end-offset {
			return nil, ErrInvalid
		}
		offset = end
	}
	if err := f.check(ctx); err != nil {
		return nil, err
	}
	return body, nil
}
