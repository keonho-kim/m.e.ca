// Purpose: Group file-browser workers, preview identity and persistent selection presentation under one owner.
// Pattern: Event-loop-owned feature state composed into the terminal session.
// Usage: File input and completion handlers mutate this state; shutdown cancels and joins its workers.
// Related: internal/tui/session/files.go, internal/tui/session/file_watch.go, internal/tui/preview/state.go.
package session

import (
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
