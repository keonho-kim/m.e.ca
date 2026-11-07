// Purpose: Persist each credential version's cryptoperiod outside the credential database format.
// Pattern: Private atomic schedule records under the maintenance directory lease.
// Usage: RotateDue reads or anchors one version, then records a confirmed replacement.
// Related: internal/storage/credentialstore/rotation.go, internal/config/statefs/files.go.
package credentialstore

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/keonho-kim/meca/internal/config/statefs"
)

const rotationDirectory = "credential-rotation"

func rotationName(ref string) string {
	sum := sha256.Sum256([]byte(ref))
	return hex.EncodeToString(sum[:])
}

func rotationDue(root *os.Root, directory string, row credentialVersion, now time.Time, period time.Duration) (due bool, resultErr error) {
	name := rotationName(row.ref)
	info, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return false, recordRotation(directory, row.ref, row.keyRef, now)
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 128 {
		return false, fmt.Errorf("rotation schedule must be a small private regular file")
	}
	file, err := root.Open(name)
	if err != nil {
		return false, err
	}
	defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
	current, err := file.Stat()
	if err != nil {
		return false, err
	}
	if !os.SameFile(info, current) || !current.Mode().IsRegular() || current.Mode().Perm()&0077 != 0 {
		return false, fmt.Errorf("rotation schedule changed while opening")
	}
	body, err := io.ReadAll(io.LimitReader(file, 129))
	if err != nil {
		return false, err
	}
	parts := strings.Split(string(body), "\n")
	if len(body) > 128 || len(parts) != 4 || parts[0] != "v1" || parts[3] != "" || len(parts[1]) != 37 || !strings.HasPrefix(parts[1], "encryption_") {
		return false, fmt.Errorf("unsupported credential rotation schedule; existing data preserved")
	}
	nanos, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || nanos <= 0 {
		return false, fmt.Errorf("invalid credential rotation time; existing data preserved")
	}
	if parts[1] != row.keyRef {
		// A foreground save or restored backup changed the credential version.
		return false, recordRotation(directory, row.ref, row.keyRef, now)
	}
	previous := time.Unix(0, nanos)
	// A backwards wall-clock adjustment must not postpone rotation indefinitely.
	return now.Before(previous) || now.Sub(previous) >= period, nil
}

func recordRotation(directory, ref, keyRef string, now time.Time) error {
	body := "v1\n" + keyRef + "\n" + strconv.FormatInt(now.UnixNano(), 10) + "\n"
	return statefs.Write(directory, rotationName(ref), []byte(body))
}
