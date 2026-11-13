// Purpose: Derive temporary large-image previews without persisting attachment originals.
// Pattern: Authenticated, pixel-budgeted media adapter.
// Usage: Browsers use this when local preview decoding would exceed its smaller pixel budget.
// Related: internal/media/thumbnail/render.go, webui/src/shared/lib/image-preview/index.ts.
package serverapi

import (
	"github.com/keonho-kim/meca/internal/agent/media"
	"github.com/keonho-kim/meca/internal/media/thumbnail"
	"net/http"
)

func (t *Transport) uiThumbnail(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := readUpload(w, r)
	if err == nil {
		body, err = thumbnail.Render(r.Context(), media.Image{Data: body})
	}
	if err != nil {
		t.writeControlReply(w, r, t.api.errorReply("", err))
		return
	}
	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(body)
}
