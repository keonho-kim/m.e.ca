// Purpose: Load and persist global non-secret TOML configuration.
// Pattern: Simple Module (configuration I/O boundary).
// Usage: Called during startup and Global override settings changes.
// Related: internal/config/statefs/files.go, internal/config/config_test.go, internal/config/settings.go.

package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/keonho-kim/v-code/internal/config/statefs"
	"github.com/pelletier/go-toml/v2"
)

const Filename = "config.toml"

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
	content, err := io.ReadAll(io.LimitReader(file, 16*1024*1024+1))
	if err != nil {
		return configuration, err
	}
	if len(content) > 16*1024*1024 {
		return configuration, fmt.Errorf("configuration exceeds 16 MiB")
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
