// Purpose: Prepare private prompt storage and load editable Markdown templates.
// Pattern: Filesystem Adapter.
// Usage: Startup publishes missing files; tasks load edits and installation explicitly refreshes defaults.
// Related: internal/config/prompts/prompts_test.go, internal/agent/prompt/defaults.go, internal/config/statefs/files.go.
package prompts

import (
	"errors"

	"io"
	"os"
	"path/filepath"

	"github.com/keonho-kim/meca/internal/agent/prompt"
	"github.com/keonho-kim/meca/internal/config/statefs"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func Bootstrap(root *os.Root) error {
	if err := prepareDirectory(root); err != nil {
		return err
	}
	for _, profile := range []prompt.Profile{prompt.Single, prompt.Team} {
		if err := prepareProfile(root, profile); err != nil {
			return err
		}
		defaults := prompt.DefaultsFor(profile)
		for name, text := range defaults.Files() {
			if err := statefs.PublishMissing(root, filepath.Join("prompts", string(profile), name), []byte(*text)); err != nil {
				return err
			}
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
		return faults.Errorf(diagcatalog.PromptsPromptsMustBeARealDirectory)
	}
	directory, err := root.Open("prompts")
	if err != nil {
		return err
	}
	return errors.Join(directory.Chmod(0700), directory.Close())
}
func Load(directory string) (prompt.Templates, error) { return LoadProfile(directory, prompt.Single) }

func LoadProfile(directory string, profile prompt.Profile) (templates prompt.Templates, err error) {
	if !profile.Valid() {
		return templates, faults.Errorf(diagcatalog.PromptsInvalidPromptProfile)
	}
	err = load(directory, profile, map[string]*string{
		"system.md": &templates.System, "development.md": &templates.Development, "compaction.md": &templates.Compaction,
	})
	return templates, err
}

func load(directory string, profile prompt.Profile, fields map[string]*string) (err error) {
	root, err := statefs.Open(directory)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	read := func(name string) (text string, err error) {
		path := filepath.Join("prompts", string(profile), name)
		info, err := root.Lstat(path)
		if err != nil {
			return "", err
		}
		if !info.Mode().IsRegular() || info.Size() > prompt.MaxTemplateBytes {
			return "", faults.Errorf(diagcatalog.PromptsMustBeARegularBoundedPromptFile, path)
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
	for name, destination := range fields {
		text, err := read(name)
		if err != nil {
			return err
		}
		*destination = text
	}
	return nil
}

func prepareProfile(root *os.Root, profile prompt.Profile) error {
	path := filepath.Join("prompts", string(profile))
	if err := root.Mkdir(path, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	info, err := root.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return faults.Errorf(diagcatalog.BootstrapMustBeARealDirectory, path)
	}
	directory, err := root.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(directory.Chmod(0700), directory.Close())
}
