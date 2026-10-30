// Purpose: Define immutable image snapshots shared by tools, model requests and persistence.
// Pattern: Provider-independent value contract.
// Usage: Image readers create snapshots; context and transport adapters retain their bytes read-only.
// Related: internal/tools/local/imagefile/decode.go, internal/agent/model/message.go.
package media

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"unicode/utf8"
)

const MaxBytes = 32 * 1024 * 1024
const MaxPixels = 32 * 1024 * 1024

// Image owns compressed bytes. Copies share Data, which consumers must never mutate.
type Image struct {
	Representative bool   `json:"representative,omitempty"`
	Path           string `json:"path"`
	MIME           string `json:"mime"`
	Format         string `json:"format"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	Digest         string `json:"digest"`
	Data           []byte `json:"data"`
}

// New creates a snapshot and takes ownership of its immutable compressed data.
func New(path, mime, format string, width, height int, data []byte) (Image, error) {
	digest := sha256.Sum256(data)
	value := Image{Path: path, MIME: mime, Format: format, Width: width, Height: height, Digest: fmt.Sprintf("%x", digest), Data: data}
	return value, value.Check()
}

// Check validates inexpensive bounds; decoding and snapshot integrity belong to their I/O boundaries.
func (i Image) Check() error {
	if i.Path == "" || len(i.Path) > 4096 || !utf8.ValidString(i.Path) || strings.ContainsRune(i.Path, 0) {
		return fmt.Errorf("invalid image path")
	}
	switch i.Format {
	case "png", "jpeg", "webp", "gif", "bmp", "tiff":
	default:
		return fmt.Errorf("unsupported image source format")
	}
	if i.MIME != "image/png" && i.MIME != "image/jpeg" && i.MIME != "image/webp" {
		return fmt.Errorf("unsupported image transport format")
	}
	if i.Width < 1 || i.Height < 1 || i.Width > MaxPixels || i.Height > MaxPixels || int64(i.Width)*int64(i.Height) > MaxPixels {
		return fmt.Errorf("image exceeds the %d-pixel decode budget", MaxPixels)
	}
	if len(i.Data) == 0 || len(i.Data) > MaxBytes {
		return fmt.Errorf("image exceeds the %d-byte limit or is empty", MaxBytes)
	}
	if len(i.Digest) != 64 || strings.Trim(i.Digest, "0123456789abcdef") != "" {
		return fmt.Errorf("invalid image digest")
	}
	return nil
}
