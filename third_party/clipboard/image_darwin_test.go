//go:build darwin && !ios

// Purpose: Verify native bounds against an isolated named pasteboard, never the user's clipboard.
// Pattern: Private AppKit integration test.
// Usage: go test -C third_party/clipboard -run TestPrivatePasteboard.
// Related: third_party/clipboard/image_darwin.go, third_party/clipboard/image_bindings_darwin.go, third_party/clipboard/LICENSE.
package clipboard

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/ebitengine/purego/objc"
)

func TestPrivatePasteboardBoundedRawData(t *testing.T) {
	loadImageNative()
	defer newAutoreleasePool()()
	if got := nsStringGo(nsString("clipboard \u2713")); got != "clipboard \u2713" {
		t.Fatal("native UTF-8 conversion", got)
	}
	pb := objc.ID(class_NSPasteboard).Send(objc.RegisterName("pasteboardWithUniqueName"))
	defer pb.Send(objc.RegisterName("releaseGlobally"))
	data := bytes.Repeat([]byte{'p'}, 128)
	pb.Send(sel_setDataForType, nsData(data), nsString("public.png"))
	value, err := readPasteboardImage(t.Context(), pb, 128)
	if err != nil || !bytes.Equal(value.Data, data) {
		t.Fatal(value, err)
	}
	if _, err := readPasteboardImage(t.Context(), pb, 127); !errors.Is(err, ErrImageLimit) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := readPasteboardImage(ctx, pb, 128); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
