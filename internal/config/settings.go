// Purpose: Define global configuration defaults and effective session policy validation.
// Pattern: Simple Module.
// Usage: Startup and settings commands use Config, Defaults and Effective.
// Related: internal/config/config.go, internal/policy/policy.go.
package config

import (
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/inference"

	"github.com/keonho-kim/v-code/internal/agent/compact"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/policy"
	"github.com/keonho-kim/v-code/internal/provider/connection"
)

type Config struct {
	ContextLength int                  `toml:"context_length"`
	Model         *SavedModel          `toml:"model,omitempty"`
	Agent         inference.Parameters `toml:"agent"`
	Diagnostics   inference.Parameters `toml:"diagnostics"`
	ModelProfiles []ModelProfile       `toml:"model_profiles,omitempty"`
	Defaults      policy.Settings      `toml:"defaults"`
	Thinking      bool                 `toml:"thinking"`
	Loop          run.Limits           `toml:"loop"`
	Compaction    Compaction           `toml:"compaction"`
	UI            UI                   `toml:"ui"`
}

// Compaction stores trigger policy and generation parameters in one TOML table.
type Compaction struct {
	compact.Settings
	inference.Parameters
}

// InferenceSettings resolves the configuration tables into runtime role parameters.
func (c Config) InferenceSettings() inference.Settings {
	return inference.Settings{Agent: c.Agent, Compaction: c.Compaction.Parameters, Diagnostic: c.Diagnostics}
}

type UI struct {
	ColumnWidth   int `toml:"column_width"`
	InputMaxLines int `toml:"input_max_lines"`
}

func Defaults() Config {
	parameters := inference.Defaults()
	return Config{
		ContextLength: connection.DefaultContextLength,
		Agent:         parameters.Agent,
		Diagnostics:   parameters.Diagnostic,
		Defaults:      policy.Settings{Permission: policy.AllowAll, Scope: policy.System},
		UI:            UI{ColumnWidth: 24, InputMaxLines: 5},
		Loop:          run.DefaultLimits(),
		Compaction:    Compaction{Settings: compact.Defaults(), Parameters: parameters.Compaction},
	}
}

func Effective(defaults, overrides policy.Settings) (policy.Settings, error) {
	if overrides.Permission != "" {
		defaults.Permission = overrides.Permission
	}
	defaults.Scope = policy.System
	return defaults, defaults.Check()
}

func check(configuration Config) error {
	if err := connection.CheckContextLength(configuration.ContextLength); err != nil {
		return err
	}
	if configuration.Model != nil {
		if _, err := configuration.Model.Selection(); err != nil {
			return fmt.Errorf("saved model: %w", err)
		}
	}
	if err := configuration.InferenceSettings().Check(); err != nil {
		return err
	}
	if err := checkProfiles(configuration.ModelProfiles); err != nil {
		return err
	}
	if err := configuration.Compaction.Settings.Check(); err != nil {
		return err
	}
	if err := configuration.Loop.Check(); err != nil {
		return err
	}
	if err := configuration.Defaults.Check(); err != nil {
		return err
	}
	if configuration.UI.ColumnWidth < 8 || configuration.UI.ColumnWidth > 80 {
		return fmt.Errorf("column_width must be between 8 and 80")
	}
	if configuration.UI.InputMaxLines < 1 || configuration.UI.InputMaxLines > 5 {
		return fmt.Errorf("input_max_lines must be between 1 and 5")
	}
	return nil
}
