// Purpose: Decode registered payload formats and verify their original length, digest and JSON.
// Pattern: Content integrity boundary.
// Usage: SQLite reads stored bytes; Codec.Restore validates them before public hydration.
// Related: internal/storage/content/compression.go, internal/storage/sqlite/payload.go.
package content

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

func ID(raw []byte) string {
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func (c *Codec) Restore(codec, size int, id string, body []byte) (json.RawMessage, error) {
	if size < 1 || size > MaxBytes {
		return nil, fmt.Errorf("invalid stored payload length")
	}
	switch codec {
	case 0:
	case 1:
		r, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		decoded, readErr := io.ReadAll(io.LimitReader(r, int64(size)+1))
		if err := errors.Join(readErr, r.Close()); err != nil {
			return nil, err
		}
		body = decoded
	case 2:
		decoded, err := c.decode(body, size)
		if err != nil {
			return nil, err
		}
		body = decoded
	default:
		return nil, fmt.Errorf("unsupported payload codec %d", codec)
	}
	if len(body) != size || ID(body) != id || !json.Valid(body) {
		return nil, fmt.Errorf("stored payload integrity check failed")
	}
	return body, nil
}
