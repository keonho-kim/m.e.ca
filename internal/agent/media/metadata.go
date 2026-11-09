// Purpose: Describe persisted image attachments without retaining or serializing their pixels.
// Pattern: Plain read projection.
// Usage: History and native APIs expose these values before an explicit image download.
// Related: internal/agent/media/image.go, internal/storage/history/record.go.
package media

type Metadata struct {
	Path, MIME, Digest string
	Width, Height      int
	Bytes              int64
}
