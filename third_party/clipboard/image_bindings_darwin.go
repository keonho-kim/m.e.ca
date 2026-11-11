//go:build darwin && !ios

// Purpose: Load upstream AppKit bindings only when image input is requested.
// Pattern: Lazy native framework binding.
// Usage: ReadImage and the upstream Init share one immutable symbol table.
// Related: third_party/clipboard/clipboard_darwin.go, third_party/clipboard/image_darwin.go, third_party/clipboard/LICENSE.
package clipboard

import (
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var nativeOnce sync.Once
var (
	appkit                   uintptr
	_NSPasteboardTypeString  uintptr
	_NSPasteboardTypePNG     uintptr
	_NSPasteboardTypeTIFF    uintptr
	class_NSPasteboard       objc.Class
	class_NSData             objc.Class
	class_NSString           objc.Class
	class_NSArray            objc.Class
	class_NSAutoreleasePool  objc.Class
	sel_alloc                objc.SEL
	sel_init                 objc.SEL
	sel_drain                objc.SEL
	sel_generalPasteboard    objc.SEL
	sel_length               objc.SEL
	sel_getBytesLength       objc.SEL
	sel_dataForType          objc.SEL
	sel_clearContents        objc.SEL
	sel_setDataForType       objc.SEL
	sel_dataWithBytesLength  objc.SEL
	sel_stringWithUTF8String objc.SEL
	sel_changeCount          objc.SEL
	sel_types                objc.SEL
	sel_count                objc.SEL
	sel_objectAtIndex        objc.SEL
	sel_UTF8String           objc.SEL
	sel_propertyListForType  objc.SEL
	sel_setPropertyListType  objc.SEL
	sel_arrayWithObjects     objc.SEL
)

func loadImageNative() {
	nativeOnce.Do(func() {
		appkit = must(purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_GLOBAL|purego.RTLD_NOW))
		_NSPasteboardTypeString = must2(purego.Dlsym(appkit, "NSPasteboardTypeString"))
		_NSPasteboardTypePNG = must2(purego.Dlsym(appkit, "NSPasteboardTypePNG"))
		_NSPasteboardTypeTIFF = must2(purego.Dlsym(appkit, "NSPasteboardTypeTIFF"))
		class_NSPasteboard = objc.GetClass("NSPasteboard")
		class_NSData = objc.GetClass("NSData")
		class_NSString = objc.GetClass("NSString")
		class_NSArray = objc.GetClass("NSArray")
		class_NSAutoreleasePool = objc.GetClass("NSAutoreleasePool")
		sel_alloc = objc.RegisterName("alloc")
		sel_init = objc.RegisterName("init")
		sel_drain = objc.RegisterName("drain")
		sel_generalPasteboard = objc.RegisterName("generalPasteboard")
		sel_length = objc.RegisterName("length")
		sel_getBytesLength = objc.RegisterName("getBytes:length:")
		sel_dataForType = objc.RegisterName("dataForType:")
		sel_clearContents = objc.RegisterName("clearContents")
		sel_setDataForType = objc.RegisterName("setData:forType:")
		sel_dataWithBytesLength = objc.RegisterName("dataWithBytes:length:")
		sel_stringWithUTF8String = objc.RegisterName("stringWithUTF8String:")
		sel_changeCount = objc.RegisterName("changeCount")
		sel_types = objc.RegisterName("types")
		sel_count = objc.RegisterName("count")
		sel_objectAtIndex = objc.RegisterName("objectAtIndex:")
		sel_UTF8String = objc.RegisterName("UTF8String")
		sel_propertyListForType = objc.RegisterName("propertyListForType:")
		sel_setPropertyListType = objc.RegisterName("setPropertyList:forType:")
		sel_arrayWithObjects = objc.RegisterName("arrayWithObjects:count:")
	})
}
