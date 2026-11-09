// Purpose: Export local clipboard pixels using macOS AppKit or existing Linux clipboard commands.
// Pattern: Concrete platform command adapters without CGO or an embedded shell.
// Usage: image.go owns the private destination and cancels or joins every command.
// Related: internal/tui/clipboard/image.go.
package clipboard

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/keonho-kim/meca/internal/agent/media"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const macImageScript = `use framework "AppKit"
use scripting additions
on run argv
  set pb to current application's NSPasteboard's generalPasteboard()
  set pixels to pb's dataForType:(current application's NSPasteboardTypePNG)
  if pixels is missing value then set pixels to pb's dataForType:(current application's NSPasteboardTypeTIFF)
  if pixels is missing value then
    set location to pb's stringForType:(current application's NSPasteboardTypeFileURL)
    if location is missing value then return "empty"
    if (location's |length|() as integer) > 8192 then return "empty"
    return "file:" & (location as text)
  end if
  if (pixels's |length|() as integer) > (item 2 of argv as integer) then return "oversized"
  if (pixels's writeToFile:(item 1 of argv) atomically:true) as boolean then return "image"
  return "failed"
end run`

func nativeImage(ctx context.Context, path string) error {
	if runtime.GOOS == "darwin" {
		cmd := exec.CommandContext(ctx, "/usr/bin/osascript", "-e", macImageScript, path, strconv.Itoa(media.MaxBytes))
		cmd.WaitDelay = time.Second
		output, err := cmd.Output()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			return faults.New(diagcatalog.Unavailable, diagcatalog.ClipboardImageReadFailed)
		}
		status := strings.TrimSpace(string(output))
		if strings.HasPrefix(status, "file:") {
			return copyClipboardFile(ctx, strings.TrimPrefix(status, "file:"), path)
		}
		if status != "image" {
			return faults.New(diagcatalog.InvalidArgument, diagcatalog.ClipboardImageUnavailable)
		}
		return nil
	}
	if runtime.GOOS != "linux" {
		return faults.New(diagcatalog.Unavailable, diagcatalog.ClipboardImageToolMissing)
	}
	name, listArgs := "xclip", []string{"-selection", "clipboard", "-target", "TARGETS", "-o"}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		name, listArgs = "wl-paste", []string{"--list-types"}
	}
	if _, err := exec.LookPath(name); err != nil {
		return faults.New(diagcatalog.Unavailable, diagcatalog.ClipboardImageToolMissing)
	}
	var types bytes.Buffer
	list := exec.CommandContext(ctx, name, listArgs...)
	list.Stdout = &imageWriter{writer: &types, limit: 16 << 10}
	list.WaitDelay = time.Second
	if err := list.Run(); err != nil {
		return faults.New(diagcatalog.InvalidArgument, diagcatalog.ClipboardImageUnavailable)
	}
	format := ""
	available := strings.Fields(types.String())
	for _, kind := range []string{"image/png", "image/jpeg", "image/webp", "image/tiff", "image/bmp", "image/gif"} {
		if slices.Contains(available, kind) {
			format = kind
			break
		}
	}
	if format == "" {
		return faults.New(diagcatalog.InvalidArgument, diagcatalog.ClipboardImageUnavailable)
	}
	args := []string{"-selection", "clipboard", "-target", format, "-o"}
	if name == "wl-paste" {
		args = []string{"--no-newline", "--type", format}
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = time.Second
	cmd.Stdout = &imageWriter{writer: file}
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return faults.New(diagcatalog.InvalidArgument, diagcatalog.ClipboardImageUnavailable)
	}
	return nil
}

type imageWriter struct {
	writer io.Writer
	bytes  int
	limit  int
}

func (w *imageWriter) Write(p []byte) (int, error) {
	limit := w.limit
	if limit == 0 {
		limit = media.MaxBytes
	}
	if len(p) > limit-w.bytes {
		return 0, faults.New(diagcatalog.InvalidArgument, diagcatalog.MediaImageDataInvalid)
	}
	n, err := w.writer.Write(p)
	w.bytes += n
	return n, err
}
