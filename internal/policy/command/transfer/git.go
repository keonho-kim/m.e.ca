// Purpose: Identify the literal destination of Git push without reading or executing repository configuration.
// Pattern: Conservative command option parser.
// Usage: The parent Git analyzer passes arguments after the push subcommand.
// Related: internal/policy/command/git.go, internal/policy/command/transfer/remote.go.
package transfer

import (
	"path/filepath"
	"strings"
)

func GitPush(args []Argument) Inspection {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !arg.Known {
			return Inspection{Indirect: true}
		}
		if arg.Value == "--" {
			args = args[i+1:]
			if len(args) == 0 || !args[0].Known {
				return Inspection{Indirect: true}
			}
			return gitDestination(args[0].Value)
		}
		if strings.HasPrefix(arg.Value, "-") {
			option, _, attached := strings.Cut(arg.Value, "=")
			switch option {
			case "--repo":
				_, value, _ := strings.Cut(arg.Value, "=")
				arg = argumentValue(args, &i, value, attached)
				if !arg.Known {
					return Inspection{Indirect: true}
				}
				return gitDestination(arg.Value)
			case "--receive-pack", "--exec":
				return Inspection{Indirect: true}
			case "-o", "--push-option":
				if !attached {
					i++
				}
			}
			continue
		}
		return gitDestination(arg.Value)
	}
	return Inspection{Indirect: true}
}

func gitDestination(value string) Inspection {
	if strings.Contains(value, "::") && !strings.Contains(value, "://") {
		return Inspection{Indirect: true} // Git transport helpers are executable code.
	}
	if filepath.IsAbs(value) || strings.HasPrefix(value, "./") || strings.HasPrefix(value, "../") || strings.HasPrefix(value, "file://") {
		return Inspection{}
	}
	if destination, remote := RemoteDestination(value); remote {
		return Inspection{Destinations: []string{destination}}
	}
	// A symbolic remote can use URL rewriting, SSH aliases and push URLs.
	// Report that uncertainty instead of inventing a network destination.
	return Inspection{Indirect: true}
}
