// Purpose: Define the supported persisted interface language choices.
// Pattern: Configuration value.
// Usage: Configuration loading and the language settings use case validate Language.
// Related: internal/config/settings.go, internal/commands/settings/language.go, internal/tui/locale/preference.go.
package config

import "fmt"

type Language string

const (
	LanguageAuto    Language = "auto"
	LanguageKorean  Language = "ko"
	LanguageEnglish Language = "en"
)

func (l Language) Check() error {
	switch l {
	case LanguageAuto, LanguageKorean, LanguageEnglish:
		return nil
	default:
		return fmt.Errorf("language must be auto, ko or en; got %q", l)
	}
}
