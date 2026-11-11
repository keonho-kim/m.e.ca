// Copyright 2021 The golang.design Initiative Authors.
// All rights reserved. Use of this source code is governed
// by a MIT license that can be found in the LICENSE file.
//
// Written by Changkun Ou <changkun.de>

//go:build darwin && !ios

// Purpose: Preserve upstream macOS write and watch operations apart from bounded MECA reads.
// Pattern: Legacy native adapter retained for dependency provenance.
// Usage: Upstream Write and Watch APIs; MECA does not invoke them.
// Related: third_party/clipboard/clipboard_darwin.go, third_party/clipboard/clipboard_pasteboard_darwin.go, third_party/clipboard/LICENSE.
package clipboard

import (
	"context"
	"time"
)

// write writes the given data to clipboard and
// returns true if success or false if failed.
// darwinItem is an Item resolved to the pasteboard type it is written under:
// either one of the built-in type objects, or a name for a custom format, which
// becomes an NSString inside the write's autorelease pool. uti names the type
// either way, and is what two items are compared on.
type darwinItem struct {
	typ  uintptr
	uti  string
	name string
	buf  []byte
	// paths is set instead of buf for FmtFiles, which is stored as a property
	// list of path strings rather than as data.
	paths []string
}

// The UTIs behind the built-in pasteboard type constants, needed as plain
// strings to notice that a custom format resolves to the same type — FmtImage
// and Register("image/png") are both public.png.
const (
	utiPlainText = "public.utf8-plain-text"
	utiPNG       = "public.png"
)

// writeAll publishes every item on one clearContents generation, so the whole
// set replaces the pasteboard together (#151). NSPasteboard is built for this:
// a generation holds as many types as it is given, and a consumer picks the
// first it understands.
func writeAll(ctx context.Context, sel selection, items []Item, loops int) (<-chan struct{}, error) {
	// loops is ignored: this platform's clipboard is a store the OS serves, so
	// no paste request ever reaches this process to be counted (see Loops).
	_ = loops
	if sel == selPrimary {
		// This platform has no primary selection. Refusing is deliberate:
		// writing to the ordinary clipboard instead would destroy whatever the
		// user had copied (see FromPrimary).
		return nil, errUnsupported
	}
	out := make([]darwinItem, 0, len(items))
	// Two different tokens can resolve to the same pasteboard type, and a second
	// store under a type would overwrite the first — inverting the rule that an
	// earlier item wins. Drop the later one instead.
	seen := make(map[string]bool, len(items))
	for _, it := range items {
		var d darwinItem
		switch it.Format {
		case FmtText:
			d = darwinItem{typ: _NSPasteboardTypeString, uti: utiPlainText, buf: it.Bytes}
		case FmtImage:
			d = darwinItem{typ: _NSPasteboardTypePNG, uti: utiPNG, buf: it.Bytes}
		case FmtFiles:
			d = darwinItem{uti: pbTypeFilenames, name: pbTypeFilenames,
				paths: pathsFromURIList(it.Bytes)}
		default:
			mime, found := formatMIME(it.Format)
			if !found {
				return nil, errUnsupported
			}
			name := darwinPasteboardTypes(mime)[0]
			d = darwinItem{uti: name, name: name, buf: it.Bytes}
		}
		if seen[d.uti] {
			continue
		}
		seen[d.uti] = true
		out = append(out, d)
	}
	if !clipboard_write_all(out) {
		return nil, errUnavailable
	}

	// use unbuffered data to prevent goroutine leak
	changed := make(chan struct{}, 1)
	cnt := clipboard_change_count()
	go func() {
		for {
			// not sure if we are too slow or the user too fast :)
			time.Sleep(time.Second)
			cur := clipboard_change_count()
			if cnt != cur {
				changed <- struct{}{}
				close(changed)
				return
			}
		}
	}()
	return changed, nil
}

func watch(ctx context.Context, sel selection, t Format) <-chan []byte {
	if sel == selPrimary {
		// No primary selection on macOS: nothing will ever be delivered, so say
		// so by closing rather than leaving the caller waiting.
		recv := make(chan []byte)
		close(recv)
		return recv
	}
	recv := make(chan []byte, 1)
	// not sure if we are too slow or the user too fast :)
	ti := time.NewTicker(time.Second)
	lastCount := clipboard_change_count()
	go func() {
		defer ti.Stop()
		for {
			select {
			case <-ctx.Done():
				close(recv)
				return
			case <-ti.C:
				this := clipboard_change_count()
				if lastCount != this {
					b, _ := Read(ctx, t, withSelection(sel)) // a failed read is nothing new to report
					if b == nil {
						continue
					}
					select {
					case recv <- b:
						lastCount = this
					case <-ctx.Done():
						close(recv)
						return
					}
				}
			}
		}
	}()
	return recv
}
