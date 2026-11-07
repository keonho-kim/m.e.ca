// Purpose: Persist a global approval mode before making it available to future requests.
// Pattern: Settings use case.
// Usage: The permission menu publishes the returned configuration after a successful save.
// Related: internal/config/settings.go, internal/policy/permission.go, internal/tui/session/permission.go.
package settings

import (
	"context"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/policy"
)

func Permission(ctx context.Context, current config.Config, value policy.Permission, update func(context.Context, func(*config.Config)) (config.Config, error)) (config.Config, error) {
	if err := value.Check(); err != nil {
		return current, err
	}
	next, err := update(ctx, func(next *config.Config) { next.Permission = value })
	if err != nil {
		return current, err
	}
	return next, nil
}
