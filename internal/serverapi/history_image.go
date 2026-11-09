// Purpose: Deliver authenticated stored raster images without base64 command envelopes.
// Pattern: HTTP binary read adapter.
// Usage: GET /v1/sessions/{id}/images/{sequence}/{index} restores one user attachment.
// Related: internal/commands/server/history_image.go, internal/serverapi/transport.go.
package serverapi

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"net/http"
	"strconv"
	"strings"
)

func (t *Transport) historyImage(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/sessions/"), "/")
	if r.Method != "GET" || len(parts) != 4 || parts[1] != "images" {
		http.NotFound(w, r)
		return
	}
	sequence, e1 := strconv.ParseInt(parts[2], 10, 64)
	index, e2 := strconv.Atoi(parts[3])
	if e1 != nil || e2 != nil {
		t.writeControlReply(w, t.api.errorReply("", faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerPageInvalid)))
		return
	}
	release, ok := t.api.readLimit().Try(1)
	if !ok {
		t.writeControlReply(w, t.api.errorReply("", faults.New(diagcatalog.ResourceExhausted, diagcatalog.ServerOriginalReadBusy)))
		return
	}
	defer release()
	read, err := t.api.Service.ReadImage(r.Context(), parts[0], sequence, index)
	if err != nil {
		t.writeControlReply(w, t.api.errorReply("", err))
		return
	}
	defer read.Release()
	w.Header().Set("Content-Type", read.Image.MIME)
	w.Header().Set("Content-Length", strconv.Itoa(len(read.Image.Data)))
	_, _ = w.Write(read.Image.Data)
}
