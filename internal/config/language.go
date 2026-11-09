// Purpose: Define the supported persisted interface language choices.
// Pattern: Configuration value.
// Usage: Configuration loading and the language settings use case validate Language.
// Related: internal/config/settings.go, internal/commands/settings/language.go, internal/tui/locale/preference.go.
package config

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

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
		return faults.Errorf(diagcatalog.ConfigLanguageMustBeAutoKoOrEn, l)
	}
}
