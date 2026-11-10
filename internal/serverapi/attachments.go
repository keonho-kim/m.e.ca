// Purpose: Accept bounded raster uploads independently of small native command envelopes.
// Pattern: Authenticated HTTP media adapter with immutable application-owned snapshots.
// Usage: Every protocol exposes /v1/attachments; StartRequest refers to returned IDs.
// Related: internal/media/imagefile/decode.go, internal/commands/server/attachments.go.
package serverapi

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/keonho-kim/meca/internal/agent/media"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/media/imagefile"
)

type attachmentValue struct {
	ExpiresAt       string `json:"expires_at"`
	ID              string `json:"id"`
	MIME            string `json:"mime"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	Bytes           int    `json:"bytes"`
	LifetimeSeconds int    `json:"lifetime_seconds"`
}

func (t *Transport) attachment(w http.ResponseWriter, r *http.Request) {
	api, release, scopeErr := t.requestAPI(r)
	if scopeErr != nil {
		t.writeControlReply(w, r, t.api.errorReply("", scopeErr))
		return
	}
	defer release()
	if r.URL.Path == "/v1/attachments" && r.Method == "POST" {
		defer r.Body.Close()
		kind, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || !strings.HasPrefix(kind, "image/") {
			t.writeControlReply(w, r, api.errorReply("", faults.New(diagcatalog.InvalidArgument, diagcatalog.MediaAttachmentType)))
			return
		}
		data, err := readUpload(w, r)
		if err != nil {
			t.writeControlReply(w, r, api.errorReply("", faults.New(diagcatalog.InvalidArgument, diagcatalog.MediaImageDataInvalid)))
			return
		}
		image, err := imagefile.Decode(r.Context(), "uploaded-image", data)
		if err != nil {
			t.writeControlReply(w, r, api.errorReply("", faults.New(diagcatalog.InvalidArgument, diagcatalog.MediaImageDataInvalid)))
			return
		}
		id, err := api.Service.StageImage(image)
		if err != nil {
			t.writeControlReply(w, r, api.errorReply("", err))
			return
		}
		t.writeAttachment(w, r, api, id, image, http.StatusCreated)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/attachments/")
	if id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}
	if r.Method == "DELETE" {
		if err := api.Service.ReleaseImage(id); err != nil {
			t.writeControlReply(w, r, api.errorReply("", err))
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method == "GET" {
		images, err := api.Service.ImageAttachments([]string{id})
		if err != nil {
			t.writeControlReply(w, r, api.errorReply("", err))
			return
		}
		t.writeAttachment(w, r, api, id, images[0], http.StatusOK)
		return
	}
	w.Header().Set("Allow", "GET, DELETE, POST")
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (t *Transport) writeAttachment(w http.ResponseWriter, r *http.Request, api *API, id string, image media.Image, status int) {
	expires := api.Service.ImageExpiry(id)
	if expires.IsZero() {
		t.writeControlReply(w, r, api.errorReply("", faults.New(diagcatalog.NotFound, diagcatalog.MediaAttachmentUnavailable)))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(attachmentValue{ExpiresAt: expires.UTC().Format(time.RFC3339Nano), ID: id, MIME: image.MIME, Width: image.Width, Height: image.Height, Bytes: len(image.Data), LifetimeSeconds: max(0, int(time.Until(expires).Seconds()))})
}

func readUpload(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	if r.ContentLength > media.MaxBytes {
		return nil, faults.New(diagcatalog.InvalidArgument, diagcatalog.MediaImageDataInvalid)
	}
	reader := http.MaxBytesReader(w, r.Body, media.MaxBytes)
	if r.ContentLength <= 0 {
		return io.ReadAll(reader)
	}
	body := make([]byte, int(r.ContentLength))
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, err
	}
	var extra [1]byte
	if _, err := reader.Read(extra[:]); err != io.EOF {
		return nil, faults.New(diagcatalog.InvalidArgument, diagcatalog.MediaImageDataInvalid)
	}
	return body, nil
}
