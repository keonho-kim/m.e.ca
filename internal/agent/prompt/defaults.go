// Purpose: Embed release Markdown templates and validate editable prompt files.
// Pattern: Simple Module (prompt assets).
// Usage: Startup publishes missing defaults, installation refreshes them and tasks load ~/.meca/prompts files.
// Related: internal/agent/prompt/defaults/system.md, internal/agent/prompt/defaults/compaction.md, internal/config/prompts/prompts.go.
package prompt

import (
	_ "embed"
	"fmt"
	"strings"
	"unicode/utf8"
)

const MaxTemplateBytes = 16 * 1024 * 1024

//go:embed defaults/system.md
var systemDefault string

//go:embed defaults/development.md
var developmentDefault string

//go:embed defaults/compaction.md
var compactionDefault string

//go:embed defaults/lat.md
var latDefault string

//go:embed defaults/lat_compaction.md
var latCompactionDefault string

type Templates struct{ System, Development, Compaction, LAT, LATCompaction string }

func Defaults() Templates {
	return Templates{System: systemDefault, Development: developmentDefault, Compaction: compactionDefault, LAT: latDefault, LATCompaction: latCompactionDefault}
}

// Files exposes the fixed prompt fields for bootstrap publication and explicit loading.
func (t *Templates) Files() map[string]*string {
	return map[string]*string{"system.md": &t.System, "development.md": &t.Development, "compaction.md": &t.Compaction, "lat.md": &t.LAT, "lat_compaction.md": &t.LATCompaction}
}

func CheckTemplate(text string) error {
	if strings.TrimSpace(text) == "" || len(text) > MaxTemplateBytes || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return fmt.Errorf("prompt must be nonempty UTF-8 Markdown within %d bytes", MaxTemplateBytes)
	}
	return nil
}
