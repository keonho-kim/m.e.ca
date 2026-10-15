// Purpose: Persist global or session Permission changes with application Scope fixed to system.
// Pattern: Settings use case.
// Usage: Terminal menus call Policy after choosing a storage destination.
// Related: internal/config/settings.go, internal/policy/policy.go.
package settings

import (
	"context"
	"github.com/keonho-kim/v-code/internal/config"
	"github.com/keonho-kim/v-code/internal/policy"
)

type PolicyStore interface {
	Settings(context.Context) (policy.Settings, error)
	SaveSettings(context.Context, policy.Settings) error
}

func Policy(ctx context.Context, store PolicyStore, saveConfig func(config.Config) error, current config.Config, value string, global bool) (config.Config, policy.Settings, error) {
	overrides, err := store.Settings(ctx)
	if err != nil {
		return current, policy.Settings{}, err
	}
	c := current
	v, err := policy.ParsePermission(value)
	if err != nil {
		return current, policy.Settings{}, err
	}
	if global {
		c.Defaults.Permission = v
	} else {
		overrides.Permission = v
	}
	if global {
		err = saveConfig(c)
	} else {
		err = store.SaveSettings(ctx, overrides)
	}
	if err != nil {
		return current, policy.Settings{}, err
	}
	current = c
	effective, err := config.Effective(c.Defaults, overrides)
	return current, effective, err
}
