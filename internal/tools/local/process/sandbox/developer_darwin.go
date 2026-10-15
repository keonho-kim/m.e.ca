//go:build darwin

// Purpose: Discover and validate the selected Apple developer installation for read-only execution.
// Pattern: macOS runtime discovery adapter.
// Usage: Prepare resolves one installation before constructing the Seatbelt profile.
// Related: internal/tools/local/process/sandbox/darwin.go, internal/tools/local/process/sandbox/environment.go.
package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func developerRuntime(ctx context.Context, project, home, private string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	canonicalHome, err := filepath.EvalSymlinks(home)
	if err != nil {
		return "", err
	}
	home = canonicalHome
	selected := os.Getenv("DEVELOPER_DIR")
	if selected == "" {
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "/usr/bin/xcode-select", "-p")
		cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C"}
		body, err := cmd.Output()
		if err != nil {
			if ctx.Err() != nil {
				return "", fmt.Errorf("developer directory discovery: %w", ctx.Err())
			}
			// Missing Apple tools must not prevent unrelated installed runtimes from working.
			return "", nil
		}
		selected = strings.TrimSpace(string(body))
	}
	if len(selected) > 4096 || strings.ContainsAny(selected, "\r\n\x00") || !filepath.IsAbs(selected) {
		return "", fmt.Errorf("invalid DEVELOPER_DIR: expected an absolute developer installation path")
	}
	if strings.HasSuffix(selected, ".app") {
		selected = filepath.Join(selected, "Contents", "Developer")
	}
	resolved, err := filepath.EvalSymlinks(selected)
	if err != nil {
		return "", fmt.Errorf("developer installation unavailable: %w", err)
	}
	if resolved == "/" || containsPath(resolved, home) || containsPath(resolved, private) || containsPath(private, resolved) || containsPath(resolved, project) || containsPath(project, resolved) {
		return "", fmt.Errorf("developer installation overlaps protected or project paths")
	}
	directory := func(name string) bool {
		info, err := os.Stat(filepath.Join(resolved, name))
		return err == nil && info.IsDir()
	}
	if !directory("usr/bin") || (!directory("SDKs") && !directory("Toolchains")) {
		return "", fmt.Errorf("DEVELOPER_DIR does not contain an Apple developer installation")
	}
	return resolved, nil
}
