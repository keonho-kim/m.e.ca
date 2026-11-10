// Purpose: Parse maintenance commands and conversation startup with a real project directory.
// Pattern: Simple Module (startup boundary).
// Usage: Called by cmd/meca before bootstrap or terminal initialization.
// Related: internal/commands/startup/parse_test.go, cmd/meca/main.go.
package startup

import (
	"errors"
	"flag"

	"io"
	"os"
	"path/filepath"
	"strings"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/distribution"
)

type Mode struct {
	Print, Export, Diagnose bool
	Server                  bool
	ServerOptions           ServerOptions
	NoShell                 bool
	Output, Prompt, Session string
	Action                  Action
	UpgradeKind             distribution.Kind
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
			return Mode{}, faults.Errorf(diagcatalog.StartupUsageMecaInstallNoShell)
		}
		return mode, nil
	}
	if len(args) > 0 && args[0] == string(Upgrade) {
		mode := Mode{Action: Upgrade, UpgradeKind: distribution.Name}
		if len(args) == 1 {
			return mode, nil
		}
		if len(args) != 2 {
			return Mode{}, faults.Errorf(diagcatalog.StartupUsageMecaUpgrade)
		}
		switch args[1] {
		case "--tui":
			mode.UpgradeKind = distribution.TUI
		case "--full":
			mode.UpgradeKind = distribution.Full
		case "--help", "-h":
			return Mode{Action: Help}, nil
		default:
			return Mode{}, faults.Errorf(diagcatalog.StartupUsageMecaUpgrade)
		}
		return mode, nil
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
	serverFlags(flags, &mode)
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return Mode{Action: Help}, nil
		}
		return Mode{}, faults.Errorf(diagcatalog.StartupUsageMecaPathPrintModeJsonSession, err)
	}
	if mode.Output != "text" && mode.Output != "json" {
		return Mode{}, faults.Errorf(diagcatalog.StartupModeMustBeTextOrJson)
	}
	if err := checkServerFlags(flags, mode); err != nil {
		return Mode{}, err
	}
	mode.Print = mode.Print || mode.Output == "json" || mode.Diagnose || mode.Export
	rest := flags.Args()
	if mode.Print {
		mode.Prompt = strings.Join(rest, " ")
	} else if len(rest) == 1 && rest[0] != "" {
		path = rest[0]
	} else if len(rest) > 0 {
		return Mode{}, faults.Errorf(diagcatalog.StartupExpectedOneProjectPath)
	}
	if mode.Export && (mode.Session == "" || mode.Prompt != "" || mode.Diagnose) {
		return Mode{}, faults.Errorf(diagcatalog.StartupExportRequiresSessionIDAndNoPrompt)
	}
	if mode.Diagnose && mode.Prompt != "" {
		return Mode{}, faults.Errorf(diagcatalog.StartupDiagnoseDoesNotAcceptAPrompt)
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
		return Mode{}, faults.Errorf(diagcatalog.StartupProjectDirectory, err)
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
		return Mode{}, faults.Errorf(diagcatalog.StartupProjectPathMustBeADirectory)
	}
	mode.Project, mode.Display = real, absolute
	return mode, nil
}
