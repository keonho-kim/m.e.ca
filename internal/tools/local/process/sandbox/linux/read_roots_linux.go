//go:build linux

// Purpose: Keep actual HOME and PATH resources visible when hosted on separate filesystem mounts.
// Pattern: Read-only runtime projection planning.
// Usage: Parent launch preparation adds explicitly needed mount roots without importing nested mounts recursively.
// Related: internal/tools/local/process/sandbox/linux/mountinfo_linux.go, internal/tools/local/process/sandbox/linux/runtime_linux.go.
package linux

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
)

func ReadRuntimeRoots(home string) ([]string, error) {
	file, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return nil, err
	}
	points, readErr := mountPoints(file)
	if err := errors.Join(readErr, file.Close()); err != nil {
		return nil, err
	}
	desired := []string{home}
	for _, path := range filepath.SplitList(os.Getenv("PATH")) {
		if !filepath.IsAbs(path) {
			continue
		}
		resolved, err := filepath.EvalSymlinks(path)
		if err == nil {
			desired = append(desired, resolved)
		}
	}
	return selectReadMounts(desired, points), nil
}
func selectReadMounts(paths, points []string) []string {
	var result []string
	for _, path := range paths {
		selected := ""
		for _, point := range points {
			if point == "/" || within("/proc", point) || within("/sys", point) || within("/run", point) || within("/dev", point) {
				continue
			}
			if within(point, path) && len(point) > len(selected) {
				selected = point
			}
		}
		if selected != "" && !slices.Contains(result, selected) {
			result = append(result, selected)
		}
	}
	return result
}
