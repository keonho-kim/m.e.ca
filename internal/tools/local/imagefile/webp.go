// Purpose: Extract one validated animated WebP frame for the standard WebP decoder.
// Pattern: RIFF container adapter; image decoding stays with golang.org/x/image.
// Usage: Decode normalizes animated WebP into a static PNG snapshot.
// Related: internal/tools/local/imagefile/decode.go.
package imagefile

import (
	"encoding/binary"
	"fmt"
)

func firstWebPFrame(data []byte) ([]byte, error) {
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" || uint64(binary.LittleEndian.Uint32(data[4:8]))+8 != uint64(len(data)) {
		return nil, fmt.Errorf("invalid animated WebP container")
	}
	for offset := 12; offset+8 <= len(data); {
		size := uint64(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		end := uint64(offset) + 8 + size
		if end > uint64(len(data)) {
			return nil, fmt.Errorf("truncated WebP chunk")
		}
		if string(data[offset:offset+4]) == "ANMF" {
			frame := data[offset+8 : int(end)]
			if len(frame) < 16 {
				return nil, fmt.Errorf("invalid WebP frame header")
			}
			width, height := frame[6:9], frame[9:12]
			chunks := frame[16:]
			// Extended header preserves alpha handling; frame coordinates do not affect
			// its own pixels. The returned snapshot describes the representative frame.
			flags := byte(0)
			if len(chunks) >= 4 && string(chunks[:4]) == "ALPH" {
				flags = 16
			}
			header := []byte{'V', 'P', '8', 'X', 10, 0, 0, 0, flags, 0, 0, 0, width[0], width[1], width[2], height[0], height[1], height[2]}
			out := make([]byte, 12+len(header)+len(chunks))
			copy(out, "RIFF")
			binary.LittleEndian.PutUint32(out[4:8], uint32(len(out)-8))
			copy(out[8:], "WEBP")
			copy(out[12:], header)
			copy(out[12+len(header):], chunks)
			return out, nil
		}
		offset = int(end + size%2)
	}
	return nil, fmt.Errorf("animated WebP contains no frame")
}
