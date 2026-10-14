// Purpose: Derive a stable readable storage group from an absolute canonical project path.
// Pattern: Pure project identity function.
// Usage: SQLite groups session files and checks their stored project against the directory identity.
// Related: internal/storage/session/session.go, internal/storage/sqlite/directories.go.
package session

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
)

func ProjectKey(project string) (string, error) {
	if err := CheckProject(project, project); err != nil {
		return "", err
	}
	name := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, filepath.Base(project))
	name = strings.Trim(name, "-_")
	if name == "" {
		name = "project"
	}
	if len(name) > 32 {
		name = name[:32]
	}
	digest := sha256.Sum256([]byte(project))
	return fmt.Sprintf("%s-%x", name, digest[:16]), nil
}
