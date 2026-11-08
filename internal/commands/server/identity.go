// Purpose: Derive stable project, task and compatible-response identities for durable deduplication.
// Pattern: Pure request identity function.
// Usage: The admission store and server Start share this identity contract.
// Related: internal/storage/serverrequests/ledger.go, internal/commands/server/tasks.go.
package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

var ErrTaskNotFound = faults.Errorf(diagcatalog.ServerRequestWasNotFound)

func TaskID(session, key string) string {
	digest := sha256.Sum256([]byte(session + "\x00" + key))
	return fmt.Sprintf("task_%x", digest[:])
}

func ProjectScope(project string) string {
	sum := sha256.Sum256([]byte(project))
	return hex.EncodeToString(sum[:])
}
func ResponseID(scope, kind, key string) string {
	sum := sha256.Sum256([]byte(scope + "\x00" + kind + "\x00" + key))
	return "resp_" + hex.EncodeToString(sum[:])
}
