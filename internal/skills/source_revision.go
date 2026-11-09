// Purpose: Invalidate name lookup reuse when installed skill directories change.
// Pattern: Filesystem identity snapshot.
// Usage: Catalog scans bind their source directories; cached name resolution rechecks them.
// Related: internal/skills/snapshot.go, internal/skills/reader.go, internal/skills/catalog_retention.go.
package skills

import (
	"errors"
	"os"
)

type sourceRevision [3]os.FileInfo

func (r *Reader) sourceRevision() (sourceRevision, bool) {
	var revision sourceRevision
	for i, source := range r.sources() {
		info, err := os.Stat(source.path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return revision, false
		}
		revision[i] = info
	}
	return revision, true
}

func sameSourceRevision(a, b sourceRevision) bool {
	for i, before := range a {
		after := b[i]
		if before == nil || after == nil {
			if before != after {
				return false
			}
			continue
		}
		if !os.SameFile(before, after) || before.Mode() != after.Mode() || !before.ModTime().Equal(after.ModTime()) || resourceChangedAt(before) != resourceChangedAt(after) {
			return false
		}
	}
	return true
}
