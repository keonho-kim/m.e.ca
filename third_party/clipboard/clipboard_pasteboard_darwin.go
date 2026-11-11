// Copyright 2021 The golang.design Initiative Authors.
// All rights reserved. Use of this source code is governed
// by a MIT license that can be found in the LICENSE file.
//
// Written by Changkun Ou <changkun.de>

//go:build darwin && !ios

// Purpose: Preserve upstream low-level pasteboard value access and conversion operations.
// Pattern: Objective-C native adapter primitives.
// Usage: Legacy platform APIs and bounded image reads share string and data primitives.
// Related: third_party/clipboard/clipboard_darwin.go, third_party/clipboard/clipboard_publish_darwin.go, third_party/clipboard/image_darwin.go, third_party/clipboard/LICENSE.
package clipboard

import (
	"bytes"
	"image/png"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"golang.org/x/image/tiff"
)

// nsdataBytes copies the contents of an NSData object into a new byte
// slice, returning nil if the object is null or empty.
func nsdataBytes(data objc.ID) []byte {
	if data == 0 {
		return nil
	}
	size := uint(data.Send(sel_length))
	if size == 0 {
		return nil
	}
	out := make([]byte, size)
	data.Send(sel_getBytesLength, unsafe.SliceData(out), size)
	runtime.KeepAlive(out)
	return out
}

func clipboard_read_string() []byte {
	defer newAutoreleasePool()()
	pasteboard := objc.ID(class_NSPasteboard).Send(sel_generalPasteboard)
	return nsdataBytes(pasteboard.Send(sel_dataForType, _NSPasteboardTypeString))
}

func clipboard_read_image() []byte {
	defer newAutoreleasePool()()
	pasteboard := objc.ID(class_NSPasteboard).Send(sel_generalPasteboard)
	if out := nsdataBytes(pasteboard.Send(sel_dataForType, _NSPasteboardTypePNG)); out != nil {
		return out
	}

	// macOS stores copied images as TIFF by default (e.g. screenshots and
	// "Copy Image" in many apps). Fall back to TIFF and transcode to PNG so
	// callers always receive PNG data, consistent with the other platforms.
	raw := nsdataBytes(pasteboard.Send(sel_dataForType, _NSPasteboardTypeTIFF))
	if raw == nil {
		return nil
	}
	img, err := tiff.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil
	}
	return buf.Bytes()
}

// clipboard_write_all clears the pasteboard once and stores every item on that
// one generation, in order — NSPasteboard treats the order types are set in as
// the order a consumer should prefer them. It reports whether every item was
// stored.
//
// The types and NSData objects are built before clearContents, so nothing
// between the clear and the last store can fail for a reason other than the
// store itself: the pasteboard is not left holding a partial set (#151).
//
// Callers pass items already deduplicated by pasteboard type (see writeAll).
func clipboard_write_all(items []darwinItem) bool {
	defer newAutoreleasePool()()

	type entry struct{ typ, data, plist objc.ID }
	entries := make([]entry, 0, len(items))
	for _, it := range items {
		typ := objc.ID(it.typ)
		if typ == 0 {
			typ = nsString(it.name)
		}
		e := entry{typ: typ}
		if it.uti == pbTypeFilenames {
			e.plist = nsStringArray(it.paths)
		} else {
			e.data = nsData(it.buf)
		}
		entries = append(entries, e)
	}

	pasteboard := objc.ID(class_NSPasteboard).Send(sel_generalPasteboard)
	pasteboard.Send(sel_clearContents)
	for _, e := range entries {
		if e.plist != 0 {
			if pasteboard.Send(sel_setPropertyListType, e.plist, e.typ) == 0 {
				return false
			}
			continue
		}
		if pasteboard.Send(sel_setDataForType, e.data, e.typ) == 0 {
			return false
		}
	}
	return true
}

// nsStringArray builds an autoreleased NSArray of NSStrings.
func nsStringArray(ss []string) objc.ID {
	ids := make([]objc.ID, len(ss))
	for i, s := range ss {
		ids[i] = nsString(s)
	}
	arr := objc.ID(class_NSArray).Send(sel_arrayWithObjects, unsafe.SliceData(ids), len(ids))
	runtime.KeepAlive(ids)
	return arr
}

// clipboard_read_filenames returns the file paths on the pasteboard, from the
// property list under NSFilenamesPboardType. macOS populates that type from the
// file URLs a modern application writes, so this reads either representation.
func clipboard_read_filenames() []string {
	defer newAutoreleasePool()()
	pasteboard := objc.ID(class_NSPasteboard).Send(sel_generalPasteboard)
	list := pasteboard.Send(sel_propertyListForType, nsString(pbTypeFilenames))
	if list == 0 {
		return nil
	}
	n := int(list.Send(sel_count))
	out := make([]string, 0, n)
	for i := range n {
		if p := nsStringGo(list.Send(sel_objectAtIndex, i)); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// nsData wraps a byte slice in an autoreleased NSData. An empty slice is passed
// as a NULL pointer, which dataWithBytes:length: accepts for a zero length.
func nsData(buf []byte) objc.ID {
	var p *byte
	if len(buf) > 0 {
		p = unsafe.SliceData(buf)
	}
	data := objc.ID(class_NSData).Send(sel_dataWithBytesLength, p, len(buf))
	runtime.KeepAlive(buf)
	return data
}

// nsString builds an autoreleased NSString from a Go string, used as a custom
// pasteboard type. It must be called inside an autorelease pool (read/write
// custom both install one) so the temporary string is reclaimed.
func nsString(s string) objc.ID {
	b := append([]byte(s), 0) // NUL-terminate for stringWithUTF8String:
	str := objc.ID(class_NSString).Send(sel_stringWithUTF8String, unsafe.SliceData(b))
	runtime.KeepAlive(b)
	return str
}

// clipboard_read_custom returns the raw bytes stored under the given MIME type,
// resolved to a pasteboard type (see darwinPasteboardTypes), or nil if no such
// data is present.
func clipboard_read_custom(mime string) []byte {
	defer newAutoreleasePool()()
	pasteboard := objc.ID(class_NSPasteboard).Send(sel_generalPasteboard)
	for _, t := range darwinPasteboardTypes(mime) {
		if out := nsdataBytes(pasteboard.Send(sel_dataForType, nsString(t))); out != nil {
			return out
		}
	}
	return nil
}

// clipboard_write_custom stores buf verbatim under the given MIME type's
// pasteboard type with no conversion (raw passthrough), replacing the current
// clipboard contents.
func clipboard_write_custom(mime string, buf []byte) bool {
	defer newAutoreleasePool()()
	pasteboard := objc.ID(class_NSPasteboard).Send(sel_generalPasteboard)
	pasteboard.Send(sel_clearContents)
	return pasteboard.Send(sel_setDataForType, nsData(buf), nsString(darwinPasteboardTypes(mime)[0])) != 0
}

func clipboard_change_count() int {
	defer newAutoreleasePool()()
	return int(objc.ID(class_NSPasteboard).Send(sel_generalPasteboard).Send(sel_changeCount))
}
