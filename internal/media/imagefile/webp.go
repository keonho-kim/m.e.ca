// Purpose: Extract one validated animated WebP frame for the standard WebP decoder.
// Pattern: RIFF container adapter; image decoding stays with golang.org/x/image.
// Usage: Decode and thumbnail.Render share representative-frame extraction.
// Related: internal/media/imagefile/decode.go, internal/media/thumbnail/render.go.
package imagefile

import (
	"bytes"
	"encoding/binary"
	"image"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

// FirstWebPFrame returns one frame after validating its dimensions against the container.
func FirstWebPFrame(data []byte, canvasWidth, canvasHeight int) ([]byte, error) {
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" || uint64(binary.LittleEndian.Uint32(data[4:8]))+8 != uint64(len(data)) {
		return nil, faults.Errorf(diagcatalog.ImagefileInvalidAnimatedWebPContainer)
	}
	for offset := 12; offset+8 <= len(data); {
		size := uint64(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		end := uint64(offset) + 8 + size
		if end > uint64(len(data)) {
			return nil, faults.Errorf(diagcatalog.ImagefileTruncatedWebPChunk)
		}
		if string(data[offset:offset+4]) == "ANMF" {
			frame := data[offset+8 : int(end)]
			if len(frame) < 16 {
				return nil, faults.Errorf(diagcatalog.ImagefileInvalidWebPFrameHeader)
			}
			width, height := frame[6:9], frame[9:12]
			read24 := func(value []byte) int { return int(value[0]) | int(value[1])<<8 | int(value[2])<<16 }
			if read24(frame[:3])*2+read24(width)+1 > canvasWidth || read24(frame[3:6])*2+read24(height)+1 > canvasHeight {
				return nil, faults.Errorf(diagcatalog.ImagefileInvalidOrExcessiveImageFrameDimensions)
			}
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
			decoded, _, err := image.DecodeConfig(bytes.NewReader(out))
			if err != nil || decoded.Width != read24(width)+1 || decoded.Height != read24(height)+1 {
				return nil, faults.Errorf(diagcatalog.ImagefileInvalidOrExcessiveImageFrameDimensions)
			}
			return out, nil
		}
		offset = int(end + size%2)
	}
	return nil, faults.Errorf(diagcatalog.ImagefileAnimatedWebPContainsNoFrame)
}
