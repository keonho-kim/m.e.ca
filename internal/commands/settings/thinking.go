// Purpose: Persist Thinking preferences with compensation across global and session storage.
// Pattern: Settings use case.
// Usage: Terminal settings applies the returned state after Thinking completes.
// Related: internal/config/config.go, internal/provider/connection/models.go.
package settings

import (
	"context"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/config"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
)

type SelectionStore interface {
	SaveSelection(context.Context, string, modelconn.Selection) error
}

func Thinking(ctx context.Context, store SelectionStore, saveConfig func(config.Config) error, session string, current config.Config, selection modelconn.Selection, value *bool, global bool) (config.Config, modelconn.Selection, error) {
	if value != nil && !selection.ThinkingSupported {
		return current, selection, fmt.Errorf("Thinking ON/OFF control is unavailable or unverified")
	}
	next := selection
	next.Thinking = value
	if value == nil {
		next.ReasoningEffort = nil
	}
	previous := current
	updated := previous
	if global {
		updated.Thinking = value
		if err := saveConfig(updated); err != nil {
			return current, selection, err
		}
	}
	if err := store.SaveSelection(ctx, session, next); err != nil {
		if global {
			if rollback := saveConfig(previous); rollback != nil {
				current = updated
				return current, selection, errors.Join(err, fmt.Errorf("restore global Thinking preference: %w", rollback))
			}
		}
		return current, selection, err
	}
	selection = next
	current = updated
	return current, selection, nil
}
