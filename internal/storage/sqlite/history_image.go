// Purpose: Decode one selected attachment directly from its inline bytes or immutable payload.
// Pattern: Bounded repository projection with image integrity verification.
// Usage: Authenticated image downloads supply a restoration memory reservation.
// Related: internal/storage/sqlite/image_metadata.go, internal/storage/sqlite/read_budget.go.
package sqlite

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/keonho-kim/meca/internal/agent/media"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/content"
	"github.com/keonho-kim/meca/internal/storage/history"
)

const historyImageRow = " FROM timeline t JOIN messages m ON t.kind='message' AND t.source_sequence=m.sequence WHERE t.sequence=? AND m.role='user'"

func historyImagePath(index int) string { return "$.images[" + strconv.Itoa(index) + "]" }

func (s *Store) HistoryImage(ctx context.Context, sequence int64, index int, bounds history.ReadBounds) (media.Image, error) {
	if index < 0 {
		return media.Image{}, fault.New(diag.NotFound, diag.ServerImageMissing)
	}
	var body, refs []byte
	err := s.db.QueryRowContext(ctx, "SELECT json_extract(m.body,?),m.refs"+historyImageRow, historyImagePath(index), sequence).Scan(&body, &refs)
	if errors.Is(err, sql.ErrNoRows) || err == nil && (len(body) == 0 || bytes.Equal(body, []byte("null"))) {
		return media.Image{}, fault.New(diag.NotFound, diag.ServerImageMissing)
	}
	if err != nil {
		return media.Image{}, err
	}
	var envelope struct {
		media.Image
		Encoded json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return media.Image{}, err
	}
	reference, err := imageDataReference(refs, index)
	if err != nil {
		return media.Image{}, err
	}
	raw := envelope.Encoded
	quoted := false
	if reference != "" {
		if !bytes.Equal(raw, []byte("null")) {
			return media.Image{}, fault.Errorf(diag.ContentInvalidStoredPayloadPlaceholder)
		}
		budget := newReadBudget(bounds.MaxBytes, func(n int) error {
			if bounds.Reserve == nil {
				return nil
			}
			return bounds.Reserve(3*n + len(body) + len(refs))
		})
		_, quoted = content.Reference(reference)
		raw, err = s.readPayload(ctx, reference, budget)
		if err != nil {
			return media.Image{}, err
		}
	}
	image := envelope.Image
	if quoted {
		// The stored string reference contains the unquoted base64 value.
		decoded := make([]byte, base64.StdEncoding.DecodedLen(len(raw)))
		n, err := base64.StdEncoding.Decode(decoded, raw)
		if err != nil {
			return media.Image{}, err
		}
		image.Data = decoded[:n]
	} else if len(raw) > 0 {
		if err := json.Unmarshal(raw, &image.Data); err != nil {
			return media.Image{}, err
		}
	}
	if err := image.Check(); err != nil {
		return media.Image{}, err
	}
	digest := sha256.Sum256(image.Data)
	if hex.EncodeToString(digest[:]) != image.Digest {
		return media.Image{}, fault.New(diag.IntegrityFailed, diag.ContentStoredPayloadIntegrityCheckFailed)
	}
	return image, nil
}

func imageDataReference(refs []byte, index int) (string, error) {
	var paths map[string]string
	if len(refs) > 0 {
		if err := json.Unmarshal(refs, &paths); err != nil {
			return "", err
		}
	}
	key := strconv.Itoa(index)
	var reference string
	for path, id := range paths {
		var parts []string
		if err := json.Unmarshal([]byte(path), &parts); err != nil {
			return "", fault.New(diag.IntegrityFailed, diag.ServerImageMissing)
		}
		if len(parts) < 2 || parts[0] != "images" || parts[1] != key {
			continue
		}
		// Valid image metadata fits in the envelope. Only data can exceed the
		// payload threshold, even when a maximum-length path is JSON escaped.
		if len(parts) != 3 || parts[2] != "data" || id == "" || reference != "" {
			return "", fault.New(diag.IntegrityFailed, diag.ServerImageMissing)
		}
		reference = id
	}
	return reference, nil
}
