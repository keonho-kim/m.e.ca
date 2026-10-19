// Purpose: Separate Git global options, subcommands and destructive actions from ordinary query arguments.
// Pattern: Command-specific argument parsing.
// Usage: Shell effect analysis passes Git calls here.
// Related: internal/policy/command/effects.go.
package command

import "strings"

func (a *analyzer) git(args []word, s *state) {
	work := s.clone()
	i := 0
	for ; i < len(args); i++ {
		arg := args[i]
		if !arg.known {
			a.result.Ask("Dynamic Git operation requires approval")
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
		if arg.text == "-c" || arg.text == "--config-env" || arg.text == "--exec-path" || arg.text == "--git-dir" || arg.text == "--work-tree" {
			a.result.Ask("Custom Git execution settings require approval")
			i++
			continue
		}
		if strings.HasPrefix(arg.text, "--exec-path=") || strings.HasPrefix(arg.text, "--config-env=") || strings.HasPrefix(arg.text, "--git-dir=") || strings.HasPrefix(arg.text, "--work-tree=") || strings.HasPrefix(arg.text, "-c") {
			a.result.Ask("Custom Git execution settings require approval")
		}
	}
	if i >= len(args) {
		return
	}
	subcommand := args[i].text
	destructive := false
	switch subcommand {
	case "reset", "clean", "restore", "checkout", "rebase", "prune", "gc", "update-ref":
		destructive = true
	case "push":
		a.result.Ask("Remote Git mutation requires approval")
	case "branch", "tag", "worktree", "stash", "reflog":
		for _, arg := range args[i+1:] {
			if !arg.known {
				destructive = true
				continue
			}
			switch arg.text {
			case "-d", "-D", "--delete", "-f", "--force", "remove", "prune", "drop", "clear", "pop", "expire", "delete":
				destructive = true
			}
		}
	}
	if destructive {
		a.result.Ask("Destructive Git operation requires approval")
		a.target(word{text: "./__git_worktree_change__", known: true}, work, false)
	}
}
