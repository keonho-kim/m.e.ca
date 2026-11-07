// Purpose: Derive stable task identities for durable request deduplication.
// Pattern: Pure request identity function.
// Usage: The admission store and server Start share this identity contract.
// Related: internal/storage/serverrequests/ledger.go, internal/commands/server/tasks.go.
package server

import (
	"crypto/sha256"
	"errors"
	"fmt"
)

var ErrTaskNotFound = errors.New("request was not found")

func TaskID(session, key string) string {
	digest := sha256.Sum256([]byte(session + "\x00" + key))
	return fmt.Sprintf("task_%x", digest[:])
}
