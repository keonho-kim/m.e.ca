// Purpose: Define session identity and project metadata independently of persistence format.
// Pattern: Simple Module.
// Usage: Storage adapters and startup use these values and boundary checks.
// Related: internal/storage/sqlite/session.go.
package session

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type Session struct{ ID, ProjectRoot, DisplayRoot string }
type Conversation struct {
	Session
	Summary, Updated string
}

func ValidID(id string) bool {
	if len(id) != 34 || !strings.HasPrefix(id, "session_") {
		return false
	}
	for _, c := range id[8:] {
		if !(c >= 'A' && c <= 'Z' || c >= '2' && c <= '7') {
			return false
		}
	}
	return true
}
func CheckProject(project, display string) error {
	if !filepath.IsAbs(project) || filepath.Clean(project) != project || len(project) > 4096 || display == "" || len(display) > 4096 || !utf8.ValidString(project+display) {
		return fmt.Errorf("invalid session project root")
	}
	return nil
}
