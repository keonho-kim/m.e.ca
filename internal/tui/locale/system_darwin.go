//go:build darwin

// Purpose: Read the macOS preferred language only when terminal locale variables are absent.
// Pattern: Native settings adapter.
// Usage: locale.System invokes this bounded read-only fallback during startup.
// Related: internal/tui/locale/system.go.
package locale

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

func nativeLanguage(ctx context.Context) string {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "/usr/bin/defaults", "read", "-g", "AppleLanguages").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(output), "\n") {
		name := strings.Trim(strings.TrimSpace(line), "\"(), ")
		if name != "" {
			return name
		}
	}
	return ""
}
