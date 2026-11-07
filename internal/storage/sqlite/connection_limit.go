// Purpose: Derive a per-connection database page ceiling from the file's immutable SQLite page size.
// Pattern: Filesystem format boundary.
// Usage: connect puts the limit in its DSN so replacement connections retain the same ceiling.
// Related: internal/storage/sqlite/store.go, internal/storage/sqlite/catalog.go.
package sqlite

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

const maxSessionBytes = 512 << 20

func sessionPageLimit(path string) (pages int64, resultErr error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
	var header [18]byte
	n, err := io.ReadFull(file, header[:])
	if n == 0 && errors.Is(err, io.EOF) {
		return maxSessionBytes / 4096, nil
	}
	if err != nil {
		return 0, err
	}
	if string(header[:16]) != "SQLite format 3\x00" {
		return 0, fmt.Errorf("invalid SQLite header; existing contents preserved")
	}
	size := int64(binary.BigEndian.Uint16(header[16:]))
	if size == 1 {
		size = 65536
	}
	if size < 512 || size > 65536 || size&(size-1) != 0 {
		return 0, fmt.Errorf("invalid SQLite page size")
	}
	return maxSessionBytes / size, nil
}
