// Purpose: Detect APNG control chunks without scanning compressed pixels or text metadata for markers.
// Pattern: Small PNG container projection; full decoding owns pixel and CRC validation.
// Usage: Decode checks PNG animation before deciding whether to reuse its compressed bytes.
// Related: internal/tools/local/imagefile/decode.go, internal/tools/local/imagefile/png_test.go.
package imagefile

import (
	"encoding/binary"
	"fmt"
)

func pngAnimation(data []byte) (bool, error) {
	// DecodeConfig has already identified the PNG signature. Chunk sizes include
	// neither their length/type header nor their CRC; subtraction avoids overflow.
	for offset := 8; offset < len(data); {
		if len(data)-offset < 12 {
			return false, fmt.Errorf("truncated PNG chunk header")
		}
		size := uint64(binary.BigEndian.Uint32(data[offset : offset+4]))
		if size > uint64(len(data)-offset-12) {
			return false, fmt.Errorf("truncated PNG chunk body")
		}
		switch string(data[offset+4 : offset+8]) {
		case "acTL":
			if size != 8 {
				return false, fmt.Errorf("invalid PNG animation control chunk")
			}
			return true, nil
		case "IDAT", "IEND":
			return false, nil
		}
		offset += int(size) + 12
	}
	return false, fmt.Errorf("PNG contains no image data")
}
