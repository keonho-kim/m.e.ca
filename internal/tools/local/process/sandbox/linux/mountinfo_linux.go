//go:build linux

// Purpose: Read bounded mount-point paths before constructing the private runtime view.
// Pattern: Kernel text protocol parser.
// Usage: enterRoot takes one mount table snapshot after making propagation private.
// Related: internal/tools/local/process/sandbox/linux/runtime_linux.go.
package linux

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

func mountPoints(reader io.Reader) ([]string, error) {
	scan := bufio.NewScanner(io.LimitReader(reader, 8*1024*1024+1))
	scan.Buffer(make([]byte, 4096), 65536)
	var points []string
	total := 0
	for scan.Scan() {
		total += len(scan.Bytes()) + 1
		if total > 8*1024*1024 || len(points) >= 65536 {
			return nil, fmt.Errorf("mount table exceeds limit")
		}
		fields := strings.Fields(scan.Text())
		if len(fields) < 10 {
			return nil, fmt.Errorf("invalid mountinfo record")
		}
		var path strings.Builder
		raw := fields[4]
		for i := 0; i < len(raw); i++ {
			if raw[i] != '\\' {
				path.WriteByte(raw[i])
				continue
			}
			if i+3 >= len(raw) {
				return nil, fmt.Errorf("invalid mount path escape")
			}
			value, err := strconv.ParseUint(raw[i+1:i+4], 8, 8)
			if err != nil || value == 0 {
				return nil, fmt.Errorf("invalid mount path escape")
			}
			path.WriteByte(byte(value))
			i += 3
		}
		decoded := path.String()
		if !filepath.IsAbs(decoded) || filepath.Clean(decoded) != decoded {
			return nil, fmt.Errorf("invalid mount point")
		}
		points = append(points, decoded)
	}
	return points, scan.Err()
}
