// Purpose: Deliver authenticated stored raster images without base64 command envelopes.
// Pattern: HTTP binary read adapter.
// Usage: GET /v1/sessions/{id}/images/{sequence}/{index} restores one user attachment.
// Related: internal/commands/server/history_image.go, internal/serverapi/transport.go.
package serverapi

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/media/thumbnail"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func (t *Transport) historyImage(w http.ResponseWriter, r *http.Request) {
	api, release, scopeErr := t.requestAPI(r)
	if scopeErr != nil {
		t.writeControlReply(w, r, t.api.errorReply("", scopeErr))
		return
	}
	defer release()
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/sessions/"), "/")
	if r.Method != "GET" || len(parts) != 4 || parts[1] != "images" {
		http.NotFound(w, r)
		return
	}
	sequence, e1 := strconv.ParseInt(parts[2], 10, 64)
	index, e2 := strconv.Atoi(parts[3])
	if e1 != nil || e2 != nil {
		t.writeControlReply(w, r, api.errorReply("", faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerPageInvalid)))
		return
	}
	query, parseErr := url.ParseQuery(r.URL.RawQuery)
	for key, values := range query {
		if key != "thumbnail" || len(values) != 1 || values[0] != "1" {
			parseErr = faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerPageInvalid)
		}
	}
	if parseErr != nil {
		t.writeControlReply(w, r, api.errorReply("", parseErr))
		return
	}
	small := query.Get("thumbnail") == "1"
	release, ok := api.readLimit().Try(1)
	if !ok {
		t.writeControlReply(w, r, api.errorReply("", faults.New(diagcatalog.ResourceExhausted, diagcatalog.ServerOriginalReadBusy)))
		return
	}
	read, err := api.Service.OpenImage(r.Context(), parts[0], sequence, index)
	release()
	if err != nil {
		t.writeControlReply(w, r, api.errorReply("", err))
		return
	}
	defer read.Release()
	var body []byte
	mime := read.Metadata.MIME
	if small {
		key := api.ScopeID + "/" + parts[0] + "/" + parts[2] + "/" + parts[3] + "/" + read.Metadata.Digest + "/" + read.Revision
		body, err = t.thumbnails.get(r.Context(), key, func() ([]byte, error) {
			release, err := api.readLimit().Acquire(r.Context(), 5)
			if err != nil {
				return nil, err
			}
			defer release()
			image, err := read.Load()
			if err != nil {
				return nil, err
			}
			return thumbnail.Render(r.Context(), image)
		})
		mime = "image/png"
	} else {
		heavyRelease, admissionErr := api.readLimit().Acquire(r.Context(), 5)
		if admissionErr != nil {
			t.writeControlReply(w, r, api.errorReply("", admissionErr))
			return
		}
		defer heavyRelease()
		image, loadErr := read.Load()
		heavyRelease()
		err = loadErr
		body = image.Data
	}
	if err != nil {
		t.writeControlReply(w, r, api.errorReply("", err))
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	_, _ = w.Write(body)
}
