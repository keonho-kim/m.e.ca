// Purpose: Copy only appended command bytes while preserving the bounded rolling-capture fallback.
// Pattern: Revision and position based read projection.
// Usage: Registry.ReadProjection reads changes under the collector lock before returning its permit.
// Related: internal/tools/local/process/output.go, internal/tools/local/process/projection_read.go.
package process

import "strings"

func (o *cappedOutput) changedBytesLocked(offset int64) int64 {
	if offset > 0 && o.written <= 2*captureHalf && offset <= o.written {
		return o.written - offset
	}
	return o.retainedBytesLocked()
}
func (o *cappedOutput) changesLocked(after uint64, offset int64) Capture {
	if o.revision == after {
		return Capture{Revision: after, Total: o.written}
	}
	if offset <= 0 || o.written > 2*captureHalf || offset > o.written {
		value := o.snapshotLocked(after)
		value.Total = o.written
		return value
	}
	originalOffset := offset
	var text strings.Builder
	text.Grow(int(o.written - offset))
	if o.spill != nil {
		body := make([]byte, int(o.written-offset))
		_, err := o.spill.file.ReadAt(body, offset)
		if err != nil {
			o.failure = err
		}
		text.Write(body)
	} else {
		if offset < int64(o.head.Len()) {
			text.Write(o.head.Bytes()[offset:])
			offset = int64(o.head.Len())
		}
		if at := offset - int64(o.head.Len()); at < int64(len(o.tail)) {
			text.Write(o.tail[at:])
		}
	}
	return Capture{Revision: o.revision, Total: o.written, Offset: originalOffset, Append: true, Text: text.String()}
}
