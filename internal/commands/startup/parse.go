// Purpose: Parse maintenance commands and conversation startup with a real project directory.
// Pattern: Simple Module (startup boundary).
// Usage: Called by cmd/v before bootstrap or terminal initialization.
// Related: internal/commands/startup/parse_test.go, cmd/v/main.go.
package startup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Mode struct {
	Action           Action
	Resume           bool
	Project, Display string
}

type Action string

const (
	Install   Action = "install"
	Uninstall Action = "uninstall"
)

func Parse(args []string, cwd string) (Mode, error) {
	if len(args) > 1 {
		return Mode{}, fmt.Errorf("usage: v [path | resume | install | uninstall]")
	}
	if len(args) == 1 && (args[0] == string(Install) || args[0] == string(Uninstall)) {
		return Mode{Action: Action(args[0])}, nil
	}
	if len(args) == 1 && args[0] == "resume" {
		return Mode{Resume: true}, nil
	}
	path := cwd
	if len(args) == 1 {
		path = args[0]
		if path == "" || strings.HasPrefix(path, "-") {
			return Mode{}, fmt.Errorf("usage: v [path | resume | install | uninstall]")
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(cwd, path)
		}
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Mode{}, err
	}
	real, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return Mode{}, fmt.Errorf("project directory: %w", err)
	}
	file, err := os.Open(real)
	if err != nil {
		return Mode{}, err
	}
	info, statErr := file.Stat()
	closeErr := file.Close()
	if statErr != nil {
		return Mode{}, statErr
	}
	if closeErr != nil {
		return Mode{}, closeErr
	}
	if !info.IsDir() {
		return Mode{}, fmt.Errorf("project path must be a directory")
	}
	return Mode{Project: real, Display: absolute}, nil
}
