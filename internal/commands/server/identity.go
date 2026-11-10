// Purpose: Derive native project, task and multimodal input identities for durable deduplication.
// Pattern: Pure request identity function.
// Usage: The admission store and server Start share this identity contract.
// Related: internal/storage/serverrequests/ledger.go, internal/commands/server/tasks.go.
package server

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/media"

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
func InputDigest(text string, images []media.Image) [32]byte {
	h := sha256.New()
	var buffer [4096]byte
	write := func(value string) {
		binary.BigEndian.PutUint64(buffer[:8], uint64(len(value)))
		_, _ = h.Write(buffer[:8])
		for len(value) > 0 {
			n := copy(buffer[:], value)
			_, _ = h.Write(buffer[:n])
			value = value[n:]
		}
	}
	write("meca-task-input-v1")
	write(text)
	binary.BigEndian.PutUint64(buffer[:8], uint64(len(images)))
	_, _ = h.Write(buffer[:8])
	for _, image := range images {
		write(image.Digest)
		write(image.MIME)
	}
	var digest [32]byte
	h.Sum(digest[:0])
	return digest
}
