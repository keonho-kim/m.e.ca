// Purpose: Initialize missing private local configuration without replacing existing state.
// Pattern: Simple Module (startup I/O boundary).
// Usage: Called before database and terminal startup by cmd/v.
// Related: internal/config/statefs/files.go, internal/config/bootstrap/bootstrap_test.go.
package bootstrap

import (
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/config"
	"github.com/keonho-kim/v-code/internal/config/prompts"
	"github.com/keonho-kim/v-code/internal/config/statefs"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

func Initialize(home string) (configuration config.Config, err error) {
	directory := filepath.Join(home, ".v")
	root, err := statefs.Open(directory)
	if err != nil {
		return configuration, err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	content, err := toml.Marshal(config.Defaults())
	if err != nil {
		return configuration, err
	}
	if err = statefs.PublishMissing(root, config.Filename, content); err != nil {
		return configuration, err
	}
	info, err := root.Lstat(config.Filename)
	if err != nil {
		return configuration, err
	}
	if !info.Mode().IsRegular() {
		return configuration, fmt.Errorf("configuration must be a regular file")
	}
	if err := root.Mkdir("logs", 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return configuration, err
	}
	info, err = root.Lstat("logs")
	if err != nil {
		return configuration, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return configuration, fmt.Errorf("logs must be a real directory")
	}
	logs, err := root.Open("logs")
	if err != nil {
		return configuration, err
	}
	if err = errors.Join(logs.Chmod(0700), logs.Close()); err != nil {
		return configuration, err
	}
	if err = prompts.Bootstrap(root); err != nil {
		return configuration, err
	}
	return config.Load(directory)
}
