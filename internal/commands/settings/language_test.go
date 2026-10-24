// Purpose: Verify language persistence failures and unsupported values preserve the current configuration.
// Pattern: Settings use-case tests.
// Usage: go test ./internal/commands/settings -run TestLanguage.
// Related: internal/commands/settings/language.go, internal/config/language.go.
package settings

import (
	"errors"
	"reflect"
	"testing"

	"github.com/keonho-kim/meca/internal/config"
)

func TestLanguageSaveFailureAndInvalidValue(t *testing.T) {
	current := config.Defaults()
	on := true
	current.Thinking = &on
	failure := errors.New("config write failed")
	calls := 0
	save := func(next config.Config) error {
		calls++
		if next.Language != config.LanguageEnglish || (next.Thinking == nil || !*next.Thinking) {
			t.Fatal("language save changed another preference")
		}
		return failure
	}
	got, err := Language(current, config.LanguageEnglish, save)
	if !errors.Is(err, failure) || !reflect.DeepEqual(got, current) || calls != 1 {
		t.Fatalf("failed save applied settings: %+v, %v", got, err)
	}
	got, err = Language(current, config.Language("unknown"), save)
	if err == nil || calls != 1 || !reflect.DeepEqual(got, current) {
		t.Fatal("unsupported language reached persistence")
	}
}
