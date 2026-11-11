//go:build darwin && !ios

// Purpose: Copy original AppKit raster data only after checking its byte size.
// Pattern: Request-scoped bounded native pasteboard read.
// Usage: ReadImage returns raw PNG/TIFF/JPEG or one local file URL without decoding pixels.
// Related: third_party/clipboard/image_bindings_darwin.go, third_party/clipboard/clipboard_darwin.go, third_party/clipboard/LICENSE.
package clipboard

import (
	"context"
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

func readImageBounded(ctx context.Context, limit int) (result Image, err error) {
	// Native framework failures must remain an unavailable input, not terminate MECA.
	defer func() {
		if recover() != nil {
			result = Image{}
			err = ErrUnavailable
		}
	}()
	loadImageNative()
	defer newAutoreleasePool()()
	pb := objc.ID(class_NSPasteboard).Send(sel_generalPasteboard)
	return readPasteboardImage(ctx, pb, limit)
}

func readPasteboardImage(ctx context.Context, pb objc.ID, limit int) (Image, error) {
	for _, name := range []string{"public.file-url", "public.png", "public.jpeg", "org.webmproject.webp", "public.webp", "public.tiff", "com.microsoft.bmp", "com.compuserve.gif"} {
		if ctx.Err() != nil {
			return Image{}, ctx.Err()
		}
		typeName := nsString(name)
		data := pb.Send(sel_dataForType, typeName)
		if data == 0 {
			continue
		}
		n := uint64(data.Send(sel_length))
		budget := uint64(limit)
		if name == "public.file-url" {
			budget = 8192
		}
		if n > budget {
			return Image{}, ErrImageLimit
		}
		if n == 0 {
			continue
		}
		buffer := make([]byte, int(n))
		data.Send(sel_getBytesLength, uintptr(unsafe.Pointer(&buffer[0])), uintptr(n))
		if ctx.Err() != nil {
			return Image{}, ctx.Err()
		}
		if name == "public.file-url" {
			return Image{FileURL: string(buffer)}, nil
		}
		return Image{Data: buffer}, nil
	}
	return Image{}, ErrNoData
}
