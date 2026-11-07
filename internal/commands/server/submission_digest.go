// Purpose: Hash normalized compatible intent without a serialized copy of the input.
// Pattern: Versioned length-prefixed digest.
// Usage: Submit binds the complete ordered request to an idempotency key.
// Related: internal/commands/server/submission.go, internal/storage/serverrequests/responses.go.
package server

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
)

func submissionDigest(input Submission) string {
	h := sha256.New()
	var buffer [4096]byte
	write := func(text string) {
		binary.BigEndian.PutUint64(buffer[:8], uint64(len(text)))
		_, _ = h.Write(buffer[:8])
		for len(text) > 0 {
			n := copy(buffer[:], text)
			_, _ = h.Write(buffer[:n])
			text = text[n:]
		}
	}
	write("meca-response-v2")
	write(input.Kind)
	write(input.Previous)
	flags := byte(0)
	if input.Stored {
		flags |= 1
	}
	if input.Background {
		flags |= 2
	}
	_, _ = h.Write([]byte{flags})
	for _, message := range input.Messages {
		write(message.Role)
		write(message.Text)
	}
	return hex.EncodeToString(h.Sum(nil))
}
