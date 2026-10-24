// Purpose: Save a global interface language preference before publishing the updated configuration.
// Pattern: Settings use case.
// Usage: The terminal language menu applies the returned configuration only after a successful save.
// Related: internal/config/language.go, internal/config/config.go, internal/tui/session/language.go.
package settings

import "github.com/keonho-kim/meca/internal/config"

func Language(current config.Config, value config.Language, saveConfig func(config.Config) error) (config.Config, error) {
	if err := value.Check(); err != nil {
		return current, err
	}
	next := current
	next.Language = value
	if err := saveConfig(next); err != nil {
		return current, err
	}
	return next, nil
}
