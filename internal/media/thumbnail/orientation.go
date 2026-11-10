// Purpose: Preserve JPEG, PNG and WebP EXIF display orientation when deriving metadata-free previews.
// Pattern: Bounded metadata parser and small-raster transform.
// Usage: Render applies orientation after downscaling so rotation never duplicates a large raster.
// Related: internal/media/thumbnail/render.go, internal/media/thumbnail/render_test.go.
package thumbnail

import (
	"bytes"
	"encoding/binary"
	"image"
)

func tiffOrientation(data []byte) int {
	if bytes.HasPrefix(data, []byte("Exif\x00\x00")) {
		data = data[6:]
	}
	if len(data) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(data[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	if order.Uint16(data[2:4]) != 42 {
		return 1
	}
	offset := int(order.Uint32(data[4:8]))
	if offset < 8 || offset > len(data)-2 {
		return 1
	}
	count := int(order.Uint16(data[offset : offset+2]))
	offset += 2
	for index := 0; index < count && offset+12 <= len(data); index++ {
		entry := data[offset : offset+12]
		offset += 12
		if order.Uint16(entry[:2]) != 0x112 || order.Uint16(entry[2:4]) != 3 || order.Uint32(entry[4:8]) != 1 {
			continue
		}
		value := int(order.Uint16(entry[8:10]))
		if value >= 1 && value <= 8 {
			return value
		}
		return 1
	}
	return 1
}
func displayOrientation(data []byte) int {
	if len(data) >= 2 && data[0] == 0xff && data[1] == 0xd8 {
		for index := 2; index < len(data); {
			if data[index] != 0xff {
				return 1
			}
			for index < len(data) && data[index] == 0xff {
				index++
			}
			if index >= len(data) {
				return 1
			}
			marker := data[index]
			index++
			if marker == 0xda || marker == 0xd9 {
				return 1
			}
			if marker == 0xd8 || marker == 1 || marker >= 0xd0 && marker <= 0xd7 {
				continue
			}
			if index+2 > len(data) {
				return 1
			}
			size := int(binary.BigEndian.Uint16(data[index : index+2]))
			if size < 2 || size > len(data)-index {
				return 1
			}
			payload := data[index+2 : index+size]
			if marker == 0xe1 && bytes.HasPrefix(payload, []byte("Exif\x00\x00")) {
				return tiffOrientation(payload)
			}
			index += size
		}
	}
	if len(data) >= 8 && bytes.Equal(data[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		for index := 8; index+12 <= len(data); {
			size := int(binary.BigEndian.Uint32(data[index : index+4]))
			if size > len(data)-index-12 {
				return 1
			}
			if string(data[index+4:index+8]) == "eXIf" {
				return tiffOrientation(data[index+8 : index+8+size])
			}
			index += size + 12
		}
	}
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		for index := 12; index+8 <= len(data); {
			size := int(binary.LittleEndian.Uint32(data[index+4 : index+8]))
			if size > len(data)-index-8 {
				return 1
			}
			if string(data[index:index+4]) == "EXIF" {
				return tiffOrientation(data[index+8 : index+8+size])
			}
			index += 8 + (size+1)&^1
		}
	}
	return 1
}
func orientPreview(source *image.NRGBA, orientation int) *image.NRGBA {
	if orientation <= 1 || orientation > 8 {
		return source
	}
	width, height := source.Bounds().Dx(), source.Bounds().Dy()
	outputWidth, outputHeight := width, height
	if orientation >= 5 {
		outputWidth, outputHeight = height, width
	}
	result := image.NewNRGBA(image.Rect(0, 0, outputWidth, outputHeight))
	for y := 0; y < outputHeight; y++ {
		for x := 0; x < outputWidth; x++ {
			sx, sy := x, y
			switch orientation {
			case 2:
				sx = width - 1 - x
			case 3:
				sx, sy = width-1-x, height-1-y
			case 4:
				sy = height - 1 - y
			case 5:
				sx, sy = y, x
			case 6:
				sx, sy = y, height-1-x
			case 7:
				sx, sy = width-1-y, height-1-x
			case 8:
				sx, sy = width-1-y, x
			}
			result.SetNRGBA(x, y, source.NRGBAAt(sx, sy))
		}
	}
	return result
}
