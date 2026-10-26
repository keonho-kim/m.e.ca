// Purpose: Persist a global approval mode before making it available to future requests.
// Pattern: Settings use case.
// Usage: The permission menu publishes the returned configuration after a successful save.
// Related: internal/config/settings.go, internal/policy/permission.go, internal/tui/session/permission.go.
package settings

import (
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/policy"
)

func Permission(current config.Config, value policy.Permission, save func(config.Config) error) (config.Config, error) {
	if err := value.Check(); err != nil {
		return current, err
	}
	next := current
	next.Permission = value
	if err := save(next); err != nil {
		return current, err
	}
	return next, nil
}
