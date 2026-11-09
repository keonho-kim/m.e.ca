// Purpose: Own diagnostic messages for user image input and attachment lifetimes.
// Pattern: Central message catalog.
// Usage: Clipboard, transport and runtime image boundaries reference these messages.
// Related: internal/agent/media/input.go, internal/serverapi/attachments.go.
package catalog

const (
	MediaImageCollectionLimit  Message = "one request accepts up to 8 images and 128 MiB of compressed image data"
	MediaImageInputUnavailable Message = "the selected model has not confirmed image input support; select a vision-capable model"
	MediaImageDataInvalid      Message = "invalid or oversized base64 image data"
	MediaAttachmentUnavailable Message = "the image attachment is missing or expired; upload it again"
	MediaAttachmentCapacity    Message = "pending image attachment capacity is occupied; release attachments or retry later"
	MediaAttachmentType        Message = "upload a raster image with an image/* Content-Type"
	ClipboardImageUnavailable  Message = "no supported image is available on the local clipboard"
	ClipboardImageReadFailed   Message = "could not read the local image clipboard"
	ClipboardImageRemote       Message = "remote terminals cannot read the local desktop clipboard; upload or attach an image file"
	ClipboardImageToolMissing  Message = "image paste requires wl-paste on Wayland or xclip on X11"
	ClipboardImagePasted       Message = "image attached"
)
