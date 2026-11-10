// Purpose: Verify authenticated thumbnail reads preserve original access and reject missing sources.
// Pattern: Native HTTP and real-storage integration test.
// Usage: go test -race ./internal/serverapi -run TestThumbnail.
// Related: internal/serverapi/history_image.go, internal/storage/sqlite/image_metadata.go.
package serverapi

import (
	"bytes"
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/media"
	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/media/imagefile"
	"github.com/keonho-kim/meca/internal/storage/history"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"image"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestThumbnailSourceValidationAndAuthentication(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	state, err := f.app.Select(t.Context(), "", true)
	if err != nil {
		t.Fatal(err)
	}
	pixels := nativePNG(t)
	original, err := imagefile.Decode(t.Context(), "input.png", pixels)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.store.Commit(t.Context(), state.Session.ID, []record.Entry{{Message: &model.Message{Role: "user", Content: "image", Images: []media.Image{original}}}}); err != nil {
		t.Fatal(err)
	}
	entries, _, err := f.store.TimelineSummaries(t.Context(), state.Session.ID, 0, 10, history.ReadBounds{})
	if err != nil || len(entries) != 1 {
		t.Fatal(entries, err)
	}
	path := fmt.Sprintf("/v1/sessions/%s/images/%d/0", state.Session.ID, entries[0].Sequence)
	transport := nativeTransport(t, f, "sse")
	for range 2 {
		reply := nativeRequest(t, transport, "GET", path+"?thumbnail=1", "")
		config, format, err := image.DecodeConfig(bytes.NewReader(reply.Body.Bytes()))
		if reply.Code != 200 || err != nil || format != "png" || config.Width != 8 || config.Height != 12 {
			t.Fatal(reply.Code, config, format, err, reply.Body.String())
		}
		if reply.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("private image became public")
		}
	}
	full := nativeRequest(t, transport, "GET", path, "")
	if full.Code != 200 || !bytes.Equal(full.Body.Bytes(), pixels) {
		t.Fatal("original changed", full.Code)
	}
	denied := httptest.NewRecorder()
	transport.ServeHTTP(denied, httptest.NewRequest("GET", "http://localhost:8787"+path+"?thumbnail=1", nil))
	if denied.Code != 401 {
		t.Fatal(denied.Code)
	}
	missing := nativeRequest(t, transport, "GET", fmt.Sprintf("/v1/sessions/%s/images/%d/1?thumbnail=1", state.Session.ID, entries[0].Sequence), "")
	if missing.Code != 404 {
		t.Fatal("missing image index was not a domain not-found", missing.Code, missing.Body.String())
	}
	if err := f.app.CloseSession(t.Context(), state.Session.ID); err != nil {
		t.Fatal(err)
	}
	catalog, err := sqlite.OpenSessions(filepath.Join(filepath.Dir(f.requests), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog.DeleteConversation(t.Context(), f.project, state.Session.ID); err != nil {
		t.Fatal(err)
	}
	removed := nativeRequest(t, transport, "GET", path+"?thumbnail=1", "")
	if removed.Code == 200 {
		t.Fatal("cached image outlived source authorization")
	}
}
