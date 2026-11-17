// Purpose: Observe Linux host and cgroup-v2 memory headroom before new external work.
// Pattern: Bounded procfs and cgroup read adapter.
// Usage: Browser admission checks Pressure on Linux and WSL.
// Related: internal/tools/browser/admission.go, internal/processmemory/pressure_test.go.
package processmemory

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func Pressure() (bool, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return false, err
	}
	available, err := availableMemory(io.LimitReader(file, 64<<10))
	closeErr := file.Close()
	if err := errors.Join(err, closeErr); err != nil {
		return false, err
	}
	// The unified mount may expose the current cgroup as its root in containers.
	directory := "/sys/fs/cgroup"
	if body, err := os.ReadFile("/proc/self/cgroup"); err == nil {
		for _, line := range strings.Split(string(body), "\n") {
			if relative, ok := strings.CutPrefix(line, "0::/"); ok {
				candidate := filepath.Join(directory, filepath.Clean("/"+relative))
				if _, err := os.Stat(filepath.Join(candidate, "memory.current")); err == nil {
					directory = candidate
				}
			}
		}
	}
	for {
		limit, lerr := os.ReadFile(filepath.Join(directory, "memory.max"))
		used, uerr := os.ReadFile(filepath.Join(directory, "memory.current"))
		if lerr == nil && uerr == nil && strings.TrimSpace(string(limit)) != "max" {
			maximum, e1 := strconv.ParseUint(strings.TrimSpace(string(limit)), 10, 64)
			current, e2 := strconv.ParseUint(strings.TrimSpace(string(used)), 10, 64)
			if e1 == nil && e2 == nil {
				if current >= maximum {
					return true, nil
				}
				available = min(available, maximum-current)
			}
		}
		if directory == "/sys/fs/cgroup" {
			break
		}
		directory = filepath.Dir(directory)
	}
	return available < 512<<20, nil
}

func availableMemory(reader io.Reader) (uint64, error) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 3 && fields[0] == "MemAvailable:" && fields[2] == "kB" {
			value, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return 0, err
			}
			if value > ^uint64(0)/1024 {
				return 0, strconv.ErrRange
			}
			return value * 1024, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, io.ErrUnexpectedEOF
}
