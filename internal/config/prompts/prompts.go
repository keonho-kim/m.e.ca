// Purpose: Prepare private prompt storage and load editable Markdown templates.
// Pattern: Filesystem Adapter.
// Usage: Startup publishes missing files; tasks load edits and installation explicitly refreshes defaults.
// Related: internal/config/prompts/prompts_test.go, internal/agent/prompt/defaults.go, internal/config/statefs/files.go.
package prompts

import (
	"errors"
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	"github.com/keonho-kim/meca/internal/config/statefs"
	"io"
	"os"
	"path/filepath"
)

func Bootstrap(root *os.Root) error {
	if err := prepareDirectory(root); err != nil {
		return err
	}
	defaults := prompt.Defaults()
	for name, text := range defaults.Files() {
		if err := statefs.PublishMissing(root, filepath.Join("prompts", name), []byte(*text)); err != nil {
			return err
		}
	}
	_, err := Load(root.Name())
	return err
}

func prepareDirectory(root *os.Root) error {
	if err := root.Mkdir("prompts", 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	info, err := root.Lstat("prompts")
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("prompts must be a real directory")
	}
	directory, err := root.Open("prompts")
	if err != nil {
		return err
	}
	return errors.Join(directory.Chmod(0700), directory.Close())
}
func Load(directory string) (templates prompt.Templates, err error) {
	root, err := statefs.Open(directory)
	if err != nil {
		return templates, err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	read := func(name string) (text string, err error) {
		path := filepath.Join("prompts", name)
		info, err := root.Lstat(path)
		if err != nil {
			return "", err
		}
		if !info.Mode().IsRegular() || info.Size() > prompt.MaxTemplateBytes {
			return "", fmt.Errorf("%s must be a regular bounded prompt file", path)
		}
		file, err := root.Open(path)
		if err != nil {
			return "", err
		}
		defer func() { err = errors.Join(err, file.Close()) }()
		body, err := io.ReadAll(io.LimitReader(file, prompt.MaxTemplateBytes+1))
		if err != nil {
			return "", err
		}
		text = string(body)
		return text, prompt.CheckTemplate(text)
	}
	for name, destination := range templates.Files() {
		text, err := read(name)
		if err != nil {
			return templates, err
		}
		*destination = text
	}
	return templates, nil
}
