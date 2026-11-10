// Purpose: Restore attachment metadata and pixels after its session leaves presentation memory.
// Pattern: Native image upload, execution and read-only history integration test.
// Usage: go test -race ./internal/serverapi -run TestStoredImage.
// Related: internal/serverapi/history_image.go, internal/storage/sqlite/history_image.go.
package serverapi

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func TestStoredImageSurvivesSessionSelectionAndUploadExpiry(t *testing.T) {
	f := newFixture(t, visionGenerate{func(_ context.Context, _ model.Request, emit func(model.Event) error) error {
		return emit(model.Event{Delta: "observed", Done: true})
	}}, false)
	transport := nativeTransport(t, f, "sse")
	pixels := nativePNG(t)
	upload := uploadImage(t, transport, pixels)
	selected := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_SelectSession{SelectSession: &v1.SelectSession{Create: true}}}).GetState()
	id := selected.GetSession().GetId()
	command := &v1.Command{Operation: &v1.Command_Start{Start: &v1.StartRequest{SessionId: id, Key: "stored-image-test", AttachmentIds: []string{upload.ID}}}}
	started := nativeCommand(t, transport, command).GetTask()
	if started == nil {
		t.Fatal("task missing")
	}
	waitNativeTask(t, deadline(t), f.app, started.GetId(), selected.GetCursor())
	if err := f.app.ReleaseImage(upload.ID); err != nil {
		t.Fatal(err)
	}
	nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_SelectSession{SelectSession: &v1.SelectSession{Create: true}}})
	history := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_History{History: &v1.HistoryRequest{SessionId: id}}})
	var sequence int64
	for _, item := range history.GetHistory().GetItems() {
		if item.GetRole() == "user" {
			if len(item.GetImages()) != 1 {
				t.Fatal(item)
			}
			sequence = item.GetSequence()
		}
	}
	if sequence < 1 {
		t.Fatal("image-only user message missing", history)
	}
	response := nativeRequest(t, transport, "GET", fmt.Sprintf("/v1/sessions/%s/images/%d/0", id, sequence), "")
	if response.Code != 200 || !bytes.Equal(response.Body.Bytes(), pixels) || response.Header().Get("Content-Type") != "image/png" {
		t.Fatal(response.Code, response.Body.String())
	}
	retry := nativeCommand(t, transport, command)
	if retry.GetTask().GetId() != started.GetId() {
		t.Fatal(retry)
	}
}
