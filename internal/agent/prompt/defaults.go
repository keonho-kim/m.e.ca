// Purpose: Embed the initial Markdown templates and validate user-owned prompt files.
// Pattern: Simple Module (prompt assets).
// Usage: Bootstrap publishes these defaults; subsequent tasks load ~/.v/prompts files.
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

//go:embed defaults/compaction.md
var compactionDefault string

//go:embed defaults/conclave.md
var conclaveDefault string

//go:embed defaults/completion.md
var completionDefault string

type Templates struct{ System, Compaction, Conclave, Completion string }

func Defaults() Templates {
	return Templates{System: systemDefault, Compaction: compactionDefault, Conclave: conclaveDefault, Completion: completionDefault}
}

// Files exposes the fixed prompt fields for bootstrap publication and explicit loading.
func (t *Templates) Files() map[string]*string {
	return map[string]*string{"system.md": &t.System, "compaction.md": &t.Compaction, "conclave.md": &t.Conclave, "completion.md": &t.Completion}
}

func CheckTemplate(text string) error {
	if strings.TrimSpace(text) == "" || len(text) > MaxTemplateBytes || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return fmt.Errorf("prompt must be nonempty UTF-8 Markdown within %d bytes", MaxTemplateBytes)
	}
	return nil
}
