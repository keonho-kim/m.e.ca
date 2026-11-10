// Purpose: Authenticate canonical image collections without allocating their base64 JSON representation.
// Pattern: Streaming deterministic encoding into a digest and byte counter.
// Usage: Private team image references use the same identity as standard JSON marshaling.
// Related: internal/storage/sqlite/team_images.go, internal/agent/media/image.go.
package sqlite

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/keonho-kim/meca/internal/agent/media"
	"io"
)

type imageCounter struct {
	writer io.Writer
	size   int
}

func (w *imageCounter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.size += n
	return n, err
}

func teamImageIdentity(images []media.Image) (string, int, error) {
	h := sha256.New()
	out := &imageCounter{writer: h}
	if images == nil {
		_, _ = io.WriteString(out, "null")
	} else {
		_, _ = io.WriteString(out, "[")
		for i, image := range images {
			if i > 0 {
				_, _ = io.WriteString(out, ",")
			}
			data := image.Data
			image.Data = nil
			metadata, err := json.Marshal(image)
			if err != nil {
				return "", 0, err
			}
			at := bytes.Index(metadata, []byte(`"data":null`)) + len(`"data":`)
			_, _ = out.Write(metadata[:at])
			_, _ = io.WriteString(out, `"`)
			encoder := base64.NewEncoder(base64.StdEncoding, out)
			if _, err := encoder.Write(data); err != nil {
				return "", 0, err
			}
			if err := encoder.Close(); err != nil {
				return "", 0, err
			}
			_, _ = io.WriteString(out, `"`)
			_, _ = out.Write(metadata[at+len("null"):])
		}
		_, _ = io.WriteString(out, "]")
	}
	return hex.EncodeToString(h.Sum(nil)), out.size, nil
}
