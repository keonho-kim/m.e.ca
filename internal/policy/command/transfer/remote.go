// Purpose: Distinguish remote uploads and remote execution from local copies and downloads.
// Pattern: Transport-specific argument adapter.
// Usage: Inspect handles scp, rsync, sftp and ssh using resolved shell words only.
// Related: internal/policy/command/transfer/inspect.go.
package transfer

import (
	"net"
	"net/url"
	"strings"
)

func inspectRemote(name string, args []Argument) Inspection {
	var result Inspection
	var values []Argument
	port, account := "", ""
	operands := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !arg.Known {
			result.Indirect = true
			values = append(values, arg)
			continue
		}
		if operands || !strings.HasPrefix(arg.Value, "-") || arg.Value == "-" {
			values = append(values, arg)
			if name == "ssh" {
				break // Remaining words are remote code, never local options.
			}
			continue
		}
		if arg.Value == "--" {
			operands = true
			continue
		}
		option, _, attached := strings.Cut(arg.Value, "=")
		result.Indirect = result.Indirect || remoteRouteOption(name, arg.Value, option)
		if remoteIdentityOption(name, arg.Value, option) {
			attachedValue := arg.Value[2:]
			if strings.HasPrefix(option, "--") {
				_, attachedValue, _ = strings.Cut(arg.Value, "=")
			}
			value := argumentValue(args, &i, attachedValue, attached || len(attachedValue) > 0)
			if !value.Known {
				result.Indirect = true
			} else if strings.HasPrefix(arg.Value, "-l") {
				account = value.Value
			} else {
				port = value.Value
			}
			continue
		}
		takes := remoteValueOption(name, option)
		if takes && !attached {
			i++
		}
	}
	if len(values) == 0 {
		return result
	}
	value := values[len(values)-1]
	if name == "ssh" || name == "sftp" {
		value = values[0]
	}
	if !value.Known {
		result.Indirect = true
		return result
	}
	if destination, remote := RemoteDestination(value.Value); remote {
		result.Destinations = append(result.Destinations, remoteIdentity(destination, port, account))
	} else if name == "ssh" || name == "sftp" {
		result.Destinations = append(result.Destinations, remoteIdentity("ssh://"+value.Value+"/", port, account))
	}
	return result
}

func remoteRouteOption(name, value, option string) bool {
	if name == "rsync" {
		return option == "--rsh" || option == "--rsync-path" || strings.HasPrefix(value, "-e")
	}
	if strings.HasPrefix(value, "-F") || strings.HasPrefix(value, "-S") || strings.HasPrefix(value, "-J") || strings.HasPrefix(value, "-o") {
		return true
	}
	return name == "ssh" && strings.ContainsAny(value[1:2], "RLDW")
}

func remoteIdentityOption(name, value, option string) bool {
	if name == "rsync" {
		return option == "--port"
	}
	if name == "ssh" {
		return strings.HasPrefix(value, "-p") || strings.HasPrefix(value, "-l")
	}
	return strings.HasPrefix(value, "-P")
}

func remoteIdentity(value, port, account string) string {
	u, err := url.Parse(value)
	if err != nil || u.Host == "" {
		return ""
	}
	if port != "" {
		u.Host = net.JoinHostPort(u.Hostname(), port)
	}
	if account != "" {
		u.User = url.User(account)
	}
	return u.String()
}

func remoteValueOption(name, option string) bool {
	if name == "rsync" {
		switch option {
		case "-e", "--rsh", "--port", "--password-file", "--files-from", "--include", "--exclude", "--include-from", "--exclude-from", "--filter", "--log-file", "--out-format", "--bwlimit", "--timeout", "--contimeout", "--rsync-path":
			return true
		}
		return false
	}
	switch option {
	case "-P", "-p", "-i", "-F", "-S", "-J", "-o", "-c", "-l", "-D", "-b", "-B", "-w", "-L", "-R", "-W":
		return option != "-p" || name == "ssh"
	}
	return false
}

// RemoteDestination translates SCP syntax without resolving SSH aliases or
// repository configuration. Unresolved routing must be explicitly reviewed.
func RemoteDestination(value string) (string, bool) {
	if strings.Contains(value, "://") {
		return value, true
	}
	colon := strings.Index(value, ":")
	if start := strings.Index(value, "["); start >= 0 {
		if end := strings.Index(value[start:], "]:"); end >= 0 {
			colon = start + end + 1
		}
	}
	if colon <= 0 || strings.Contains(value[:colon], "/") {
		return "", false
	}
	host, path := value[:colon], value[colon+1:]
	if strings.HasPrefix(path, ":") {
		return "rsync://" + host + "/" + strings.TrimPrefix(path, ":"), true
	}
	if !strings.HasPrefix(path, "/") {
		path = "/~/" + path // Relative to the remote account's home, not /.
	}
	u, err := url.Parse("ssh://" + host)
	if err != nil {
		return "", true
	}
	u.Path = path
	return u.String(), true
}
