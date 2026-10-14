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
)

type Config struct {
	Model         *SavedModel        `toml:"model,omitempty"`
	Inference     inference.Settings `toml:"inference"`
	ModelProfiles []ModelProfile     `toml:"model_profiles,omitempty"`
	Defaults      policy.Settings    `toml:"defaults"`
	Thinking      bool               `toml:"thinking"`
	Loop          run.Limits         `toml:"loop"`
	Compaction    compact.Settings   `toml:"compaction"`
	UI            UI                 `toml:"ui"`
	Session       Session            `toml:"session"`
}

type UI struct {
	Theme         string `toml:"theme"`
	ColumnWidth   int    `toml:"column_width"`
	InputMaxLines int    `toml:"input_max_lines"`
}

type Session struct {
	RestoreLast bool `toml:"restore_last"`
}

func Defaults() Config {
	return Config{
		Inference:  inference.Defaults(),
		Defaults:   policy.Settings{Permission: policy.Approve, Scope: policy.Directory},
		UI:         UI{Theme: "classic", ColumnWidth: 24, InputMaxLines: 5},
		Session:    Session{RestoreLast: true},
		Loop:       run.DefaultLimits(),
		Compaction: compact.Defaults(),
	}
}

func Effective(defaults, overrides policy.Settings) (policy.Settings, error) {
	if overrides.Permission != "" {
		defaults.Permission = overrides.Permission
	}
	if overrides.Scope != "" {
		defaults.Scope = overrides.Scope
	}
	return defaults, defaults.Check()
}

func check(configuration Config) error {
	if configuration.Model != nil {
		if _, err := configuration.Model.Selection(); err != nil {
			return fmt.Errorf("saved model: %w", err)
		}
	}
	if err := configuration.Inference.Check(); err != nil {
		return err
	}
	if err := checkProfiles(configuration.ModelProfiles); err != nil {
		return err
	}
	if err := configuration.Compaction.Check(); err != nil {
		return err
	}
	if err := configuration.Loop.Check(); err != nil {
		return err
	}
	if err := configuration.Defaults.Check(); err != nil {
		return err
	}
	if configuration.UI.Theme != "classic" {
		return fmt.Errorf("only the classic theme is supported")
	}
	if configuration.UI.ColumnWidth < 8 || configuration.UI.ColumnWidth > 80 {
		return fmt.Errorf("column_width must be between 8 and 80")
	}
	if configuration.UI.InputMaxLines < 1 || configuration.UI.InputMaxLines > 5 {
		return fmt.Errorf("input_max_lines must be between 1 and 5")
	}
	return nil
}
