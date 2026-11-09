// Purpose: Bound durable admission rows across projects without expiring deduplication identities.
// Pattern: Serialized admission with a persistent conservative allocation ceiling.
// Usage: Admit reserve only genuinely new request records.
// Related: internal/storage/serverrequests/ledger.go, internal/storage/capacity/reserve.go.
package serverrequests

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/keonho-kim/meca/internal/storage/capacity"
)

const maxAdmissions = 100000

func (l *Ledger) reserve(ctx context.Context) (*capacity.Reservation, error) {
	return capacity.Reserve(ctx, filepath.Join(l.directory, "capacity"), maxAdmissions, func() (count int64, resultErr error) {
		file, err := os.Open(l.directory)
		if err != nil {
			return 0, err
		}
		defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
		for {
			if err := ctx.Err(); err != nil {
				return 0, err
			}
			entries, err := file.ReadDir(128)
			for _, entry := range entries {
				if entry.IsDir() && len(entry.Name()) == 64 {
					if _, err := hex.DecodeString(entry.Name()); err != nil {
						continue
					}
					path := filepath.Join(l.directory, entry.Name(), Filename)
					u := url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro"}
					db, err := sql.Open("sqlite", u.String())
					if err != nil {
						return 0, err
					}
					var rows int64
					readErr := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM requests").Scan(&rows)
					closeErr := db.Close()
					if err := errors.Join(readErr, closeErr); err != nil {
						return 0, err
					}
					count += rows
				}
				if strings.HasSuffix(entry.Name(), ".json") {
					count++
					if count >= maxAdmissions {
						return 0, capacity.ErrLimit
					}
				}
			}
			if errors.Is(err, io.EOF) {
				return count, nil
			}
			if err != nil {
				return 0, err
			}
		}
	})
}
