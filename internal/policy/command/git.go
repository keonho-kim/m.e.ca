// Purpose: Separate Git global options, subcommands and destructive actions from ordinary query arguments.
// Pattern: Command-specific argument parsing.
// Usage: Shell effect analysis passes Git calls here.
// Related: internal/policy/command/effects.go.
package command

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

func (a *analyzer) git(args []word, s *state) {
	work := s.clone()
	i := 0
	for ; i < len(args); i++ {
		arg := args[i]
		if !arg.known {
			a.ask("Dynamic Git operation requires approval")
			return
		}
		if !strings.HasPrefix(arg.text, "-") {
			break
		}
		if arg.text == "-C" {
			if i+1 >= len(args) {
				return
			}
			i++
			a.cd(args[i:i+1], work)
			continue
		}
		if strings.HasPrefix(arg.text, "-C") && len(arg.text) > 2 {
			a.cd([]word{{text: arg.text[2:], known: true}}, work)
			continue
		}
		if arg.text == "-c" {
			if i+1 >= len(args) {
				a.ask("Custom Git execution settings require approval")
				return
			}
			i++
			if !args[i].known || executableGitConfig(args[i].text) {
				a.gitExecutionConfig(args[i], work)
			}
			continue
		}
		if arg.text == "--config-env" || arg.text == "--exec-path" || arg.text == "--git-dir" || arg.text == "--work-tree" {
			a.ask("Custom Git execution settings require approval")
			i++
			continue
		}
		if strings.HasPrefix(arg.text, "--exec-path=") || strings.HasPrefix(arg.text, "--config-env=") || strings.HasPrefix(arg.text, "--git-dir=") || strings.HasPrefix(arg.text, "--work-tree=") || strings.HasPrefix(arg.text, "-c") && executableGitConfig(arg.text[2:]) {
			a.gitExecutionConfig(word{text: arg.text[2:], known: true}, work)
		}
	}
	if i >= len(args) {
		return
	}
	subcommand := args[i].text
	destructive := false
	switch subcommand {
	case "reset", "restore", "rebase", "prune", "gc", "update-ref":
		destructive = true
	case "clean", "rm", "mv":
		destructive = unknownGitOption(args[i+1:]) || !gitOption(args[i+1:], "-n", "--dry-run") || gitOption(args[i+1:], "--no-dry-run")
	case "checkout":
		destructive = unknownGitOption(args[i+1:]) || !gitOption(args[i+1:], "-b") || gitOption(args[i+1:], "-f", "--force", "-B")
	case "push":
		a.ask("Remote Git mutation requires approval")
	case "branch", "tag":
		destructive = unknownGitOption(args[i+1:]) || gitOption(args[i+1:], "-d", "-D", "--delete", "-f", "--force")
	case "worktree", "stash", "reflog":
		if i+1 < len(args) {
			action := args[i+1]
			destructive = !action.known || action.text == "remove" || action.text == "prune" || action.text == "drop" || action.text == "clear" || action.text == "pop" || action.text == "expire" || action.text == "delete"
		}

	}
	if destructive {
		a.ask("Destructive Git operation requires approval")
		a.directoryTarget(word{text: ".", known: true}, work, false)
	}
}

func executableGitConfig(value string) bool {
	name, _, ok := strings.Cut(strings.ToLower(value), "=")
	if !ok {
		return true
	}
	for _, prefix := range []string{"alias.", "filter.", "include.", "includeif."} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	switch name {
	case "core.sshcommand", "core.gitproxy", "core.fsmonitor", "core.hookspath", "core.pager", "credential.helper", "diff.external", "interactive.difffilter":
		return true
	}
	return strings.HasPrefix(name, "pager.") || strings.HasSuffix(name, ".command") || strings.HasSuffix(name, ".textconv")
}

func gitOption(args []word, flags ...string) bool {
	for _, arg := range args {
		if !arg.known {
			continue
		}
		if arg.text == "--" {
			break
		}
		for _, flag := range flags {
			if arg.text == flag {
				return true
			}
			if len(flag) == 2 && strings.HasPrefix(flag, "-") && strings.HasPrefix(arg.text, "-") && !strings.HasPrefix(arg.text, "--") && strings.Contains(arg.text[1:], flag[1:]) {
				return true
			}
		}
	}
	return false
}

func unknownGitOption(args []word) bool {
	for _, arg := range args {
		if arg.known && arg.text == "--" {
			break
		}
		if !arg.known {
			return true
		}
	}
	return false
}

func (a *analyzer) gitExecutionConfig(value word, s *state) {
	a.ask("Custom Git execution settings require approval")
	if !value.known {
		return
	}
	key, code, ok := strings.Cut(value.text, "=")
	if ok && strings.HasPrefix(strings.ToLower(key), "alias.") && strings.HasPrefix(code, "!") {
		a.parse(code[1:], syntax.LangPOSIX, s.clone())
	}
}
