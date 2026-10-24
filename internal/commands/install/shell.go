// Purpose: Add or remove only MECA-owned PATH blocks in shell configuration.
// Pattern: Shell configuration adapter.
// Usage: Install plans registrations; Uninstall removes exact recorded blocks.
// Related: internal/commands/install/state.go, internal/commands/install/install.go.
package install

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const pathBlock = "# >>> meca PATH >>>\ncase \":$PATH:\" in\n  *\":$HOME/.local/bin:\"*) ;;\n  *) export PATH=\"$HOME/.local/bin:$PATH\" ;;\nesac\n# <<< meca PATH <<<\n"

type profileChange struct {
	registration
	body    []byte
	mode    os.FileMode
	before  []byte
	existed bool
}

func validProfile(name string) bool {
	switch name {
	case ".zshrc", ".bashrc", ".bash_profile", ".bash_login", ".profile":
		return true
	}
	return false
}

func profileNames(root *os.Root, shell string) ([]string, error) {
	switch filepath.Base(shell) {
	case "zsh":
		return []string{".zshrc"}, nil
	case "bash":
		for _, name := range []string{".bash_profile", ".bash_login"} {
			if _, err := root.Lstat(name); err == nil {
				return []string{".bashrc", name}, nil
			} else if !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
		}
		return []string{".bashrc", ".profile"}, nil
	case "sh", "dash":
		return []string{".profile"}, nil
	default:
		return nil, fmt.Errorf("automatic PATH registration supports zsh, bash and sh; SHELL=%q", shell)
	}
}

func planProfiles(root *os.Root, names []string, previous []registration, remove bool) ([]profileChange, error) {
	records := map[string]registration{}
	for _, r := range previous {
		records[r.Name] = r
	}
	var changes []profileChange
	for _, name := range names {
		body, exists, mode, err := readFile(root, name)
		if err != nil {
			return nil, err
		}
		r, known := records[name]
		if !known {
			r = registration{Name: name, Created: !exists, Separator: len(body) > 0 && body[len(body)-1] != '\n'}
		}
		text := string(body)
		count := strings.Count(text, pathBlock)
		if count > 1 || strings.Contains(text, "# >>> meca PATH >>>") && count == 0 {
			return nil, fmt.Errorf("%s contains an edited MECA PATH block; restore or remove that block first", name)
		}
		block := pathBlock
		if r.Separator {
			block = "\n" + block
		}
		if remove {
			if !exists || count == 0 {
				continue
			}
			if !strings.Contains(text, block) {
				return nil, fmt.Errorf("%s registration boundary changed", name)
			}
			text = strings.Replace(text, block, "", 1)
		} else if count == 0 {
			text += block
		}
		changes = append(changes, profileChange{r, []byte(text), mode, body, exists})
	}
	return changes, nil
}

func writeProfile(root *os.Root, change profileChange, remove bool) (err error) {
	current, exists, mode, err := readFile(root, change.Name)
	if err != nil {
		return err
	}
	if exists != change.existed || mode != change.mode || !bytes.Equal(current, change.before) {
		return fmt.Errorf("%s changed during installation; retry with the current file", change.Name)
	}
	if remove && change.Created && len(change.body) == 0 {
		return root.Remove(change.Name)
	}
	name := ".meca-profile-" + rand.Text()
	file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, change.mode)
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := root.Remove(name); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}()
	_, writeErr := file.Write(change.body)
	if err = errors.Join(writeErr, file.Sync(), file.Close()); err != nil {
		return err
	}
	return root.Rename(name, change.Name)
}
