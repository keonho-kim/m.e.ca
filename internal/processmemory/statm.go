// Purpose: Convert the resident-page field of a proc statm snapshot into bytes.
// Pattern: Pure input-boundary function.
// Usage: The Linux RSS reader supplies its bounded snapshot and native page size.
// Related: internal/processmemory/resident_linux.go, internal/processmemory/statm_test.go.
package processmemory

import (
	"bytes"

	"strconv"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func residentPages(snapshot []byte, pageSize uint64) (uint64, error) {
	if pageSize == 0 {
		return 0, faults.Errorf(diagcatalog.ProcessmemoryInvalidResidentMemoryPageSize)
	}
	column := 0
	for field := range bytes.FieldsSeq(snapshot) {
		if column == 1 {
			pages, err := strconv.ParseUint(string(field), 10, 64)
			if err != nil {
				return 0, faults.Errorf(diagcatalog.ProcessmemoryReadResidentPageCount, err)
			}
			if pages > ^uint64(0)/pageSize {
				return 0, faults.Errorf(diagcatalog.ProcessmemoryResidentMemorySizeOverflows)
			}
			return pages * pageSize, nil
		}
		column++
	}
	return 0, faults.Errorf(diagcatalog.ProcessmemoryResidentPageCountIsMissing)
}
