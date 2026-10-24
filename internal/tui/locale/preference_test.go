// Purpose: Verify explicit language preferences override the host locale and auto follows it.
// Pattern: Deterministic environment boundary tests.
// Usage: go test ./internal/tui/locale -run TestLanguagePreference.
// Related: internal/tui/locale/preference.go, internal/config/language.go.
package locale

import (
	"context"
	"testing"

	"github.com/keonho-kim/meca/internal/config"
)

func TestLanguagePreference(t *testing.T) {
	t.Setenv("LC_ALL", "C")
	if Resolve(context.Background(), config.LanguageKorean) != Korean || Resolve(context.Background(), config.LanguageAuto) != English {
		t.Fatal("explicit Korean did not override the host locale")
	}
	t.Setenv("LC_ALL", "ko_KR.UTF-8")
	t.Setenv("LANGUAGE", "")
	if Resolve(context.Background(), config.LanguageEnglish) != English || Resolve(context.Background(), config.LanguageAuto) != Korean {
		t.Fatal("explicit English or automatic locale selection failed")
	}
}
