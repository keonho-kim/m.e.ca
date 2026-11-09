// Purpose: Define and validate global model, execution and presentation defaults.
// Pattern: Simple Module.
// Usage: Startup and local settings commands use Config and Defaults.
// Related: internal/config/config.go, internal/config/language.go, internal/agent/inference/settings.go.
package config

import (
	"github.com/keonho-kim/meca/internal/agent/inference"

	"github.com/keonho-kim/meca/internal/agent/compact"
	"github.com/keonho-kim/meca/internal/agent/run"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

type Config struct {
	Permission    policy.Permission    `toml:"permission"`
	Language      Language             `toml:"language"`
	ContextLength int                  `toml:"context_length"`
	Model         *SavedModel          `toml:"model,omitempty"`
	Agent         inference.Parameters `toml:"agent"`
	Diagnostics   inference.Parameters `toml:"diagnostics"`
	Thinking      *bool                `toml:"thinking,omitempty"`
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
		Language:      LanguageAuto,
		ContextLength: connection.DefaultContextLength,
		Agent:         parameters.Agent,
		Diagnostics:   parameters.Diagnostic,
		UI:            UI{ColumnWidth: 24, InputMaxLines: 5},
		Loop:          run.DefaultLimits(),
		Compaction:    Compaction{Settings: compact.Defaults(), Parameters: parameters.Compaction},
	}
}

func check(configuration Config) error {
	if err := configuration.Permission.Check(); err != nil {
		return err
	}
	if err := configuration.Language.Check(); err != nil {
		return err
	}
	if err := connection.CheckContextLength(configuration.ContextLength); err != nil {
		return err
	}
	if configuration.Model != nil {
		if _, err := configuration.Model.Selection(); err != nil {
			return faults.Errorf(diagcatalog.ConfigSavedModel, err)
		}
	}
	if err := configuration.InferenceSettings().Check(); err != nil {
		return err
	}
	if err := configuration.Compaction.Settings.Check(); err != nil {
		return err
	}
	if err := configuration.Loop.Check(); err != nil {
		return err
	}
	if configuration.UI.ColumnWidth < 8 || configuration.UI.ColumnWidth > 80 {
		return faults.Errorf(diagcatalog.ConfigColumnWidthMustBeBetweenAnd)
	}
	if configuration.UI.InputMaxLines < 1 || configuration.UI.InputMaxLines > 5 {
		return faults.Errorf(diagcatalog.ConfigInputMaxLinesMustBeBetweenAnd)
	}
	return nil
}
