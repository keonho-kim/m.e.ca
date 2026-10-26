// Purpose: Parse maintenance commands and conversation startup with a real project directory.
// Pattern: Simple Module (startup boundary).
// Usage: Called by cmd/meca before bootstrap or terminal initialization.
// Related: internal/commands/startup/parse_test.go, cmd/meca/main.go.
package startup

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Mode struct {
	Print, Export, Diagnose bool
	NoShell                 bool
	Output, Prompt, Session string
	Action                  Action
	Resume                  bool
	Project, Display        string
}

type Action string

const (
	Help      Action = "help"
	Install   Action = "install"
	Uninstall Action = "uninstall"
	Upgrade   Action = "upgrade"
)

func Parse(args []string, cwd string) (Mode, error) {
	if len(args) > 0 && args[0] == string(Install) {
		mode := Mode{Action: Install}
		flags := flag.NewFlagSet("meca install", flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		flags.BoolVar(&mode.NoShell, "no-shell", false, "install without activating a login shell")
		if err := flags.Parse(args[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return Mode{Action: Help}, nil
			}
			return Mode{}, err
		}
		if flags.NArg() != 0 {
			return Mode{}, fmt.Errorf("usage: meca install [--no-shell]")
		}
		return mode, nil
	}
	if len(args) > 0 && args[0] == string(Upgrade) {
		if len(args) != 1 {
			return Mode{}, fmt.Errorf("usage: meca upgrade")
		}
		return Mode{Action: Upgrade}, nil
	}
	if len(args) == 1 && args[0] == string(Uninstall) {
		return Mode{Action: Action(args[0])}, nil
	}
	if len(args) == 1 && args[0] == "resume" {
		return Mode{Resume: true}, nil
	}
	mode := Mode{Output: "text"}
	path := cwd
	if len(args) > 1 && !strings.HasPrefix(args[0], "-") && strings.HasPrefix(args[1], "-") {
		path, args = args[0], args[1:]
	}
	flags := flag.NewFlagSet("meca", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.BoolVar(&mode.Print, "print", false, "run once and print the final response")
	flags.StringVar(&mode.Output, "mode", "text", "text or json")
	flags.StringVar(&mode.Session, "session", "", "resume a session ID in this project")
	flags.BoolVar(&mode.Export, "export", false, "export the selected session as JSONL")
	flags.BoolVar(&mode.Diagnose, "diagnose", false, "explicitly diagnose the selected model")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return Mode{Action: Help}, nil
		}
		return Mode{}, fmt.Errorf("%w; usage: meca [path] [--print|--mode json] [--session ID] [prompt]", err)
	}
	if mode.Output != "text" && mode.Output != "json" {
		return Mode{}, fmt.Errorf("mode must be text or json")
	}
	mode.Print = mode.Print || mode.Output == "json" || mode.Diagnose || mode.Export
	rest := flags.Args()
	if mode.Print {
		mode.Prompt = strings.Join(rest, " ")
	} else if len(rest) == 1 && rest[0] != "" {
		path = rest[0]
	} else if len(rest) > 0 {
		return Mode{}, fmt.Errorf("expected one project path")
	}
	if mode.Export && (mode.Session == "" || mode.Prompt != "" || mode.Diagnose) {
		return Mode{}, fmt.Errorf("--export requires --session ID and no prompt or diagnostic")
	}
	if mode.Diagnose && mode.Prompt != "" {
		return Mode{}, fmt.Errorf("--diagnose does not accept a prompt")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
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
	mode.Project, mode.Display = real, absolute
	return mode, nil
}
