// Purpose: Embed profile-specific Markdown guidance and validate editable prompt files.
// Pattern: Simple Module (prompt assets).
// Usage: Startup publishes missing defaults, installation refreshes them and tasks load the selected ~/.meca/prompts profile.
// Related: internal/agent/prompt/defaults/system.md, internal/agent/prompt/defaults/team.md, internal/agent/prompt/defaults/team-development.md, internal/config/prompts/prompts.go.
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

//go:embed defaults/team-development.md
var teamDevelopmentDefault string

//go:embed defaults/compaction.md
var compactionDefault string

//go:embed defaults/team.md
var teamSupplement string

var teamDefault = systemDefault + "\n" + teamSupplement

type Profile string

const (
	Single Profile = "single"
	Team   Profile = "team"
)

func (p Profile) Valid() bool { return p == Single || p == Team }

type Templates struct {
	System, Development, Compaction string
}

func Defaults() Templates { return DefaultsFor(Single) }

func DefaultsFor(profile Profile) Templates {
	system, development := systemDefault, developmentDefault
	if profile == Team {
		system, development = teamDefault, teamDevelopmentDefault
	}
	return Templates{
		System: system, Development: development, Compaction: compactionDefault,
	}
}

// Files exposes the fixed prompt fields for bootstrap publication and explicit loading.
func (t *Templates) Files() map[string]*string {
	return map[string]*string{
		"system.md": &t.System, "development.md": &t.Development, "compaction.md": &t.Compaction,
	}
}

func CheckTemplate(text string) error {
	if strings.TrimSpace(text) == "" || len(text) > MaxTemplateBytes || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return fmt.Errorf("prompt must be nonempty UTF-8 Markdown within %d bytes", MaxTemplateBytes)
	}
	return nil
}
