// Purpose: Decode registered payload formats and verify their original length, digest and JSON.
// Pattern: Content integrity boundary.
// Usage: SQLite restores JSON; detail archives verify compressed blocks before page reads.
// Related: internal/storage/content/compression.go, internal/storage/sqlite/payload.go.
package content

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"io"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func ID(raw []byte) string {
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func (c *Codec) DecodeVerified(codec, size int, id string, body []byte) ([]byte, error) {
	if size < 1 || size > MaxBytes {
		return nil, faults.Errorf(diagcatalog.ContentInvalidStoredPayloadLength)
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
		return nil, faults.Errorf(diagcatalog.ContentUnsupportedPayloadCodec, codec)
	}
	if len(body) != size || ID(body) != id {
		return nil, faults.Errorf(diagcatalog.ContentStoredPayloadIntegrityCheckFailed)
	}
	return body, nil
}

func (c *Codec) Restore(codec, size int, id string, body []byte) (json.RawMessage, error) {
	raw, err := c.DecodeVerified(codec, size, id, body)
	if err != nil {
		return nil, err
	}
	if !json.Valid(raw) {
		return nil, faults.Errorf(diagcatalog.ContentStoredPayloadIntegrityCheckFailed)
	}
	return raw, nil
}
