// Purpose: Define session identity and project metadata independently of persistence format.
// Pattern: Simple Module.
// Usage: Storage adapters and startup use these values and boundary checks.
// Related: internal/storage/sqlite/session.go.
package session

import (
	"path/filepath"
	"strings"
	"unicode/utf8"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"

	// ErrNotFound distinguishes a missing conversation from an unreadable or unsupported database.
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

var ErrNotFound = faults.Errorf(diagcatalog.SessionSessionWasNotFound)
var ErrForeignSession = faults.Errorf(diagcatalog.SessionSessionDoesNotBelongToThisDatabase)
var ErrCursor = faults.Errorf(diagcatalog.SessionInvalidOrExpiredSessionPageCursorRefresh)
var ErrPageLimit = faults.Errorf(diagcatalog.SessionSessionListSnapshotLimitReachedRefreshRetry)

type Session struct{ ID, ProjectRoot, DisplayRoot string }
type Conversation struct {
	Session
	Summary, Updated string
	Unavailable      string
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
		return faults.Errorf(diagcatalog.SessionInvalidSessionProjectRoot)
	}
	return nil
}
