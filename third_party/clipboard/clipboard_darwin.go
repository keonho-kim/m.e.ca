// Copyright 2021 The golang.design Initiative Authors.
// All rights reserved. Use of this source code is governed
// by a MIT license that can be found in the LICENSE file.
//
// Written by Changkun Ou <changkun.de>

//go:build darwin && !ios

// Purpose: Map macOS pasteboard formats and own thread-local native call lifetimes.
// Pattern: Upstream platform adapter with lazy framework initialization.
// Usage: Legacy clipboard APIs and MECA bounded image reads share native primitives.
// Related: third_party/clipboard/image_bindings_darwin.go, third_party/clipboard/clipboard_pasteboard_darwin.go, third_party/clipboard/clipboard_publish_darwin.go.
package clipboard

import (
	"context"
	"runtime"
	"strings"
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

// pbTypeFilenames is NSFilenamesPboardType, the pasteboard type holding a file
// list as a property-list array of paths. It is spelled out rather than looked
// up with dlsym because the constant's value is exactly this string, and the
// symbol is deprecated.
//
// It is a legacy type, but it is the one that reads and writes a *whole* list in
// a single call: NSPasteboardTypeFileURL holds one URL per pasteboard item, and
// dataForType: only ever sees the first. macOS bridges the two — writing this
// property list makes the system synthesize public.file-url items, and after
// another application writes NSURLs this type returns their paths — so one call
// interoperates in both directions (see specs/file-clipboard.md §3).
const pbTypeFilenames = "NSFilenamesPboardType"

func must(sym uintptr, err error) uintptr {
	if err != nil {
		panic(err)
	}
	return sym
}

func must2(sym uintptr, err error) uintptr {
	if err != nil {
		panic(err)
	}
	// dlsym returns a pointer to the object so dereference like this to avoid possible misuse of 'unsafe.Pointer' warning
	return **(**uintptr)(unsafe.Pointer(&sym))
}

func initialize() error {
	loadImageNative()
	return nil
}

// newAutoreleasePool creates an NSAutoreleasePool and returns a function that
// drains it. Every pasteboard operation runs inside one: accessors such as
// -dataForType: and +[NSData dataWithBytes:length:] return autoreleased
// objects, but these goroutines run on arbitrary OS threads with no pool of
// their own, so without draining the objects leak — notably in the per-second
// poll loops driven by write and watch. Use as: defer newAutoreleasePool()().
//
// An autorelease pool is thread-local and must be drained on the same OS
// thread it was created on. A goroutine can otherwise migrate threads between
// creation and drain (e.g. across the allocations in the TIFF transcode),
// which crashes when the pool is popped on the wrong thread. Pin the OS thread
// for the pool's lifetime to keep creation and drain together.
func newAutoreleasePool() (drain func()) {
	runtime.LockOSThread()
	pool := objc.ID(class_NSAutoreleasePool).Send(sel_alloc).Send(sel_init)
	return func() {
		pool.Send(sel_drain)
		runtime.UnlockOSThread()
	}
}

// enumerateFormats reports the formats currently on the clipboard by reading the
// general pasteboard's advertised types and mapping each to a Format.
func enumerateFormats(ctx context.Context, sel selection) []Format {
	if sel == selPrimary {
		return nil
	}
	defer newAutoreleasePool()()
	pasteboard := objc.ID(class_NSPasteboard).Send(sel_generalPasteboard)
	types := pasteboard.Send(sel_types)
	if types == 0 {
		return nil
	}
	n := int(objc.ID(types).Send(sel_count))
	out := make([]Format, 0, n)
	for i := 0; i < n; i++ {
		t := objc.ID(types).Send(sel_objectAtIndex, uintptr(i))
		if f, ok := darwinFormatFor(nsStringGo(objc.ID(t))); ok {
			out = append(out, f)
		}
	}
	return out
}

// darwinNativeTypes aliases a portable MIME type to the pasteboard type other
// macOS applications publish that data under. Pasteboard types are UTIs — their
// own namespace, not MIME types — so using the MIME string verbatim only ever
// round-trips with this library itself, never with another app (#160).
//
// Only aliases whose data is the MIME type's bytes verbatim belong here, since
// custom formats are raw passthrough.
var darwinNativeTypes = map[string]string{
	"text/html":       "public.html",
	"application/pdf": "com.adobe.pdf",
	"text/rtf":        "public.rtf",
	"image/png":       "public.png",
	"image/tiff":      "public.tiff",
	"image/jpeg":      "public.jpeg",
}

// darwinPasteboardTypes returns the pasteboard types a MIME type may appear
// under, most preferred first: its native UTI (when it has one), then the MIME
// string itself. Reads try each in turn, so data published under either is
// reachable; writes use the first.
func darwinPasteboardTypes(mime string) []string {
	if uti, ok := darwinNativeTypes[mime]; ok {
		return []string{uti, mime}
	}
	return []string{mime}
}

// darwinMIMEForType is the inverse of darwinNativeTypes: it maps a pasteboard
// UTI back to the MIME type it stands for.
func darwinMIMEForType(t string) (string, bool) {
	for mime, uti := range darwinNativeTypes {
		if t == uti {
			return mime, true
		}
	}
	return "", false
}

// darwinFormatFor maps a pasteboard type (a UTI, or for a MIME type without a
// native alias the MIME string used verbatim) to a Format: the built-in
// text/image UTIs to FmtText/FmtImage, an aliased UTI to its MIME type's custom
// token, and any other MIME-shaped type to a custom format registered on demand.
// The built-in image UTIs are matched first, so public.png/public.tiff report
// FmtImage even though they also alias image/png and image/tiff; both tokens
// read the same bytes.
func darwinFormatFor(t string) (Format, bool) {
	switch t {
	case "public.utf8-plain-text", "public.plain-text", "NSStringPboardType":
		return FmtText, true
	case "public.png", "public.tiff":
		return FmtImage, true
	case pbTypeFilenames, "public.file-url":
		return FmtFiles, true
	}
	if mime, ok := darwinMIMEForType(t); ok {
		return Register(mime), true
	}
	if strings.Contains(t, "/") {
		return Register(t), true
	}
	return 0, false
}

// nsStringGo converts an NSString to a Go string via its UTF8String pointer.
func nsStringGo(s objc.ID) string {
	if s == 0 {
		return ""
	}
	return objc.Send[string](s, sel_UTF8String)
}

func read(ctx context.Context, sel selection, t Format) (buf []byte, err error) {
	if sel == selPrimary {
		// This platform has no primary selection (see FromPrimary).
		return nil, errUnsupported
	}
	switch t {
	case FmtFiles:
		paths := clipboard_read_filenames()
		if len(paths) == 0 {
			// No file list on the pasteboard is the ordinary empty case, not a
			// pasteboard this process cannot reach.
			return nil, ErrNoData
		}
		return uriListFromPaths(paths), nil
	case FmtText:
		return clipboard_read_string(), nil
	case FmtImage:
		return clipboard_read_image(), nil
	default:
		mime, ok := formatMIME(t)
		if !ok {
			return nil, errUnsupported
		}
		return clipboard_read_custom(mime), nil
	}
}

// sensitive reports whether the pasteboard carries one of the nspasteboard.org
// markers for sensitive content (see Sensitive).
func sensitive(ctx context.Context, sel selection) (bool, error) {
	if sel == selPrimary {
		return false, errUnsupported // no primary selection on macOS
	}
	defer newAutoreleasePool()()
	pasteboard := objc.ID(class_NSPasteboard).Send(sel_generalPasteboard)
	types := pasteboard.Send(sel_types)
	if types == 0 {
		return false, nil
	}
	n := int(objc.ID(types).Send(sel_count))
	for i := 0; i < n; i++ {
		t := objc.ID(types).Send(sel_objectAtIndex, uintptr(i))
		if containsAny([]string{nsStringGo(objc.ID(t))}, darwinSensitiveTypes...) {
			return true, nil
		}
	}
	return false, nil
}
