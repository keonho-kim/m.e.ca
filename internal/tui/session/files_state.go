// Purpose: Own browser navigation, preview and selection transitions with revision-bound read slots.
// Pattern: Event-loop-owned feature state composed into the terminal session.
// Usage: Session owns I/O and delegates accepted value transitions to this event-loop state.
// Related: internal/tui/session/files.go, internal/tui/session/file_watch.go, internal/tui/preview/state.go.
package session

import (
	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/keonho-kim/meca/internal/tui/browser"
	"github.com/keonho-kim/meca/internal/tui/browser/readqueue"
	"github.com/keonho-kim/meca/internal/tui/preview"
)

type fileBrowserState struct {
	browser                      browser.Browser
	directoryReads               readqueue.Slot[directoryPage]
	directoryRevision            int
	directoryLoading             bool
	fileWatch                    fileWatch
	watchReads                   readqueue.Slot[watchRequest]
	watchRevision                int
	fileSelectionBusy            bool
	fileSelectionQueue           []string
	fileSendConfirm, fileReserve bool
	referenceRevision            uint64
	references                   referenceProjection
	filePreview                  *preview.State
	filePreviewPath              string
	previewReads                 readqueue.Slot[previewRequest]
	previewRevision              int
	previewLayouts               readqueue.Slot[previewLayoutRequest]
	previewLayoutRevision        int
	previewLayoutDesired         preview.Snapshot
}

func (f *fileBrowserState) cancelDirectory() {
	f.directoryReads.Stop()
	f.directoryRevision++
	f.directoryLoading = false
}

func (f *fileBrowserState) planDirectory(mode, path string, offset int) (directoryPage, bool) {
	f.cancelDirectory()
	f.directoryLoading = mode != "refresh"
	page := directoryPage{revision: f.directoryRevision, mode: mode, path: path, offset: offset}
	return page, f.directoryReads.Submit(page)
}

func (f *fileBrowserState) applyDirectory(page *directoryPage) {
	switch page.mode {
	case "open":
		f.browser.Open(page.path, page.entries, page.more)
	case "push":
		f.browser.Push(page.path, page.entries, page.more)
	case "page":
		if column := f.browser.Current(); column != nil && column.Path == page.path {
			column.Entries, column.More, column.Offset, column.Index = page.entries, page.more, page.offset, 0
		}
	case "refresh":
		if column := f.browser.Current(); column != nil && column.Path == page.path {
			selected := ""
			if column.Index >= 0 && column.Index < len(column.Entries) {
				selected = column.Entries[column.Index].Name
			}
			column.Entries, column.More = page.entries, page.more
			column.Index = max(0, min(column.Index, len(page.entries)-1))
			for i, entry := range page.entries {
				if entry.Name == selected {
					column.Index = i
					break
				}
			}
		}
	}
}

func (f *fileBrowserState) beginSelection(path string) bool {
	if f.fileSelectionBusy {
		f.fileSelectionQueue = append(f.fileSelectionQueue, path)
		return false
	}
	f.fileSelectionBusy = true
	return true
}

func (f *fileBrowserState) completeSelection(result *fileSelectionResult) (string, bool) {
	f.fileSelectionBusy = false
	if result.err == nil {
		f.referenceRevision++
		if f.browser.Selected == nil {
			f.browser.Selected = map[string]bool{}
		}
		if result.selected {
			f.browser.Selected[result.path] = true
		} else {
			delete(f.browser.Selected, result.path)
		}
	}
	if len(f.fileSelectionQueue) > 0 {
		path := f.fileSelectionQueue[0]
		f.fileSelectionQueue[0] = ""
		f.fileSelectionQueue = f.fileSelectionQueue[1:]
		return path, true
	}
	f.fileSelectionQueue = nil
	return "", false
}

func (f *fileBrowserState) applyPreview(revision int, result *fileview.Preview, failure error, redact func(string) string) {
	if f.filePreview == nil || revision != f.previewRevision {
		return
	}
	f.filePreview.Loading = false
	if failure != nil {
		f.filePreview.Error = redact(failure.Error())
		return
	}
	if result != nil {
		f.filePreview.Path = result.Path
		f.filePreview.Content = result.Content
		f.filePreview.Truncated = result.Truncated
	}
}
