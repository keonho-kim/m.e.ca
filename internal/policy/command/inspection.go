// Purpose: Select commands whose arguments need detailed safety or shell-state analysis.
// Pattern: Fixed inspection dispatch.
// Usage: Calls resolve ordinary command names and scan only executable substitutions in their arguments.
// Related: internal/policy/command/calls.go, internal/policy/command/effects.go, internal/policy/command/wrappers.go.
package command

// This is the sole command inspection list; all other resolved executables use the default path.
func inspectionRequired(name string) bool {
	switch name {
	case "sudo", "su", "doas", "pkexec", "cd", "eval", ".", "source", "unset", "export", "readonly", "local", "set", "read", "getopts", "shift", "alias", "unalias", "trap",
		"sh", "dash", "bash", "zsh", "ksh", "ash", "env", "command", "exec", "builtin", "nohup", "setsid", "nice", "timeout", "stdbuf", "time", "busybox", "toybox", "xargs",
		"rm", "rmdir", "unlink", "mv", "cp", "install", "ln", "mkdir", "touch", "truncate", "tee", "chmod", "chown", "chgrp", "find", "git", "sed", "gsed",
		"curl", "wget", "scp", "rsync", "sftp", "ssh",
		"dd", "mkfs", "mke2fs", "fdisk", "sfdisk", "parted", "wipefs", "mount", "umount", "chroot":
		return true
	default:
		return false
	}
}
