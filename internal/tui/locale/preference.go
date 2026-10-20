// Purpose: Resolve a configured UI language before consulting the host locale.
// Pattern: Presentation policy.
// Usage: Conversation selection, session startup and language changes call Resolve.
// Related: internal/config/language.go, internal/tui/locale/system.go, internal/tui/session/language.go.
package locale

import (
	"context"

	"github.com/keonho-kim/v-code/internal/config"
)

func Resolve(ctx context.Context, preference config.Language) Language {
	switch preference {
	case config.LanguageKorean:
		return Korean
	case config.LanguageEnglish:
		return English
	default:
		return System(ctx)
	}
}
