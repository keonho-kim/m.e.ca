// Purpose: Read a stable bounded binary reference through the verified skill-file boundary.
// Pattern: Asset I/O boundary.
// Usage: The existing local raster decoder consumes ResourceBytes for skill images.
// Related: internal/skills/resources.go, internal/tools/local/skill_image.go.
package skills

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func validResourcePath(path string) bool {
	return filepath.IsLocal(path) && !strings.ContainsRune(path, 0) && len(path) <= 4096
}

func (r *Reader) resourceRoot(ctx context.Context, id, path string) (*os.Root, string, error) {
	if !validResourcePath(path) {
		return nil, "", ErrInvalid
	}
	dir, scope, err := r.resolve(ctx, instruction.Selector{ID: id})
	if err != nil {
		return nil, "", err
	}
	root, actual, err := r.openRoot(ctx, dir)
	if err != nil {
		return nil, "", err
	}
	doc, err := r.documentAt(ctx, root, dir, scope, actual, true)
	if err != nil {
		return nil, "", errors.Join(err, root.Close())
	}
	if doc.ManualOnly && !instruction.ExplicitRead(ctx, doc.ID) && !instruction.ExplicitRead(ctx, id) {
		return nil, "", errors.Join(ErrManual, root.Close())
	}
	return root, doc.ID, nil
}
