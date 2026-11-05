// Purpose: Convert the resident-page field of a proc statm snapshot into bytes.
// Pattern: Pure input-boundary function.
// Usage: The Linux RSS reader supplies its bounded snapshot and native page size.
// Related: internal/processmemory/resident_linux.go, internal/processmemory/statm_test.go.
package processmemory

import (
	"bytes"
	"fmt"
	"strconv"
)

func residentPages(snapshot []byte, pageSize uint64) (uint64, error) {
	if pageSize == 0 {
		return 0, fmt.Errorf("invalid resident-memory page size")
	}
	column := 0
	for field := range bytes.FieldsSeq(snapshot) {
		if column == 1 {
			pages, err := strconv.ParseUint(string(field), 10, 64)
			if err != nil {
				return 0, fmt.Errorf("read resident page count: %w", err)
			}
			if pages > ^uint64(0)/pageSize {
				return 0, fmt.Errorf("resident-memory size overflows")
			}
			return pages * pageSize, nil
		}
		column++
	}
	return 0, fmt.Errorf("resident page count is missing")
}
