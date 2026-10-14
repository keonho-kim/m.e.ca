// Purpose: Persist global or session Scope and Permission changes and compute effective policy.
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

func Policy(ctx context.Context, store PolicyStore, saveConfig func(config.Config) error, current config.Config, field, value string, global bool) (config.Config, policy.Settings, error) {
	overrides, err := store.Settings(ctx)
	if err != nil {
		return current, policy.Settings{}, err
	}
	c := current
	if field == "permission" {
		v, err := policy.ParsePermission(value)
		if err != nil {
			return current, policy.Settings{}, err
		}
		if global {
			c.Defaults.Permission = v
		} else {
			overrides.Permission = v
		}
	} else {
		v, err := policy.ParseScope(value)
		if err != nil {
			return current, policy.Settings{}, err
		}
		if global {
			c.Defaults.Scope = v
		} else {
			overrides.Scope = v
		}
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
