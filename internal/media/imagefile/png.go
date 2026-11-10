// Purpose: Detect APNG control chunks without scanning compressed pixels or text metadata for markers.
// Pattern: Small PNG container projection; full decoding owns pixel and CRC validation.
// Usage: Decode checks PNG animation before deciding whether to reuse its compressed bytes.
// Related: internal/media/imagefile/decode.go, internal/media/imagefile/png_test.go.
package imagefile

import (
	"encoding/binary"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"

	// DecodeConfig has already identified the PNG signature. Chunk sizes include
	// neither their length/type header nor their CRC; subtraction avoids overflow.
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func pngAnimation(data []byte) (bool, error) {

	for offset := 8; offset < len(data); {
		if len(data)-offset < 12 {
			return false, faults.Errorf(diagcatalog.ImagefileTruncatedPNGChunkHeader)
		}
		size := uint64(binary.BigEndian.Uint32(data[offset : offset+4]))
		if size > uint64(len(data)-offset-12) {
			return false, faults.Errorf(diagcatalog.ImagefileTruncatedPNGChunkBody)
		}
		switch string(data[offset+4 : offset+8]) {
		case "acTL":
			if size != 8 {
				return false, faults.Errorf(diagcatalog.ImagefileInvalidPNGAnimationControlChunk)
			}
			return true, nil
		case "IDAT", "IEND":
			return false, nil
		}
		offset += int(size) + 12
	}
	return false, faults.Errorf(diagcatalog.ImagefilePNGContainsNoImageData)
}
