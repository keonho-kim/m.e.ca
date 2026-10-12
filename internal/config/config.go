// Purpose: Load and persist global non-secret TOML configuration.
// Pattern: Simple Module (configuration I/O boundary).
// Usage: Called during startup and Global override settings changes.
// Related: internal/config/statefs/files.go, internal/config/config_test.go, internal/policy/policy.go.

package config

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/config/statefs"
	"io"
	"os"

	"github.com/keonho-kim/v-code/internal/agent/compact"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/policy"
	"github.com/pelletier/go-toml/v2"
)

const Filename = "config.toml"

type Config struct {
	Defaults   policy.Settings  `toml:"defaults"`
	Thinking   bool             `toml:"thinking"`
	Loop       run.Limits       `toml:"loop"`
	Compaction compact.Settings `toml:"compaction"`
	UI         UI               `toml:"ui"`
	Session    Session          `toml:"session"`
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
		Defaults:   policy.Settings{Permission: policy.Approve, Scope: policy.Directory},
		UI:         UI{Theme: "classic", ColumnWidth: 24, InputMaxLines: 5},
		Session:    Session{RestoreLast: true},
		Loop:       run.DefaultLimits(),
		Compaction: compact.Defaults(),
	}
}

func Load(directory string) (configuration Config, err error) {
	configuration = Defaults()
	root, err := statefs.Open(directory)
	if err != nil {
		return configuration, err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	file, err := root.Open(Filename)
	if errors.Is(err, os.ErrNotExist) {
		return configuration, nil
	}
	if err != nil {
		return configuration, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	content, err := io.ReadAll(io.LimitReader(file, 64*1024+1))
	if err != nil {
		return configuration, err
	}
	if len(content) > 64*1024 {
		return configuration, fmt.Errorf("configuration exceeds 64 KiB")
	}
	if err = toml.NewDecoder(bytes.NewReader(content)).DisallowUnknownFields().Decode(&configuration); err != nil {
		return configuration, fmt.Errorf("parse configuration: %w", err)
	}
	return configuration, check(configuration)
}

func Save(directory string, configuration Config) error {
	if err := check(configuration); err != nil {
		return err
	}
	content, err := toml.Marshal(configuration)
	if err != nil {
		return err
	}
	return statefs.Write(directory, Filename, content)
}

func Effective(defaults, project policy.Settings) (policy.Settings, error) {
	if project.Permission != "" {
		defaults.Permission = project.Permission
	}
	if project.Scope != "" {
		defaults.Scope = project.Scope
	}
	return defaults, defaults.Check()
}

func check(configuration Config) error {
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
