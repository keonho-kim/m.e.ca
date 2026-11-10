// Purpose: Deliver native uploaded image snapshots to a vision model and reject unsupported selections.
// Pattern: Native HTTP, engine and immutable snapshot integration tests.
// Usage: go test -race ./internal/serverapi -run TestVisionInput.
// Related: internal/serverapi/attachments.go, api/meca/v1/control.proto.
package serverapi

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"net/http/httptest"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

type visionGenerate struct{ generateFunc }

func (visionGenerate) ImageInputSupported() bool { return true }

func uploadImage(t *testing.T, transport *Transport, data []byte) attachmentValue {
	t.Helper()
	r := httptest.NewRequestWithContext(deadline(t), "POST", "http://localhost:8787/v1/attachments", bytes.NewReader(data))
	r.Header.Set("Authorization", "Bearer "+testToken)
	r.Header.Set("Content-Type", "image/png")
	w := httptest.NewRecorder()
	transport.ServeHTTP(w, r)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var value attachmentValue
	if err := json.Unmarshal(w.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	return value
}
func nativePNG(t *testing.T) []byte {
	t.Helper()
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewGray(image.Rect(0, 0, 8, 12))); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}
func TestVisionInputAcrossNativeProtocols(t *testing.T) {
	for _, protocol := range []string{"sse", "socket", "grpc"} {
		t.Run(protocol, func(t *testing.T) {
			pixels := nativePNG(t)
			seen := make(chan model.Request, 1)
			f := newFixture(t, visionGenerate{func(_ context.Context, r model.Request, emit func(model.Event) error) error {
				seen <- r
				return emit(model.Event{Delta: "image received", Done: true})
			}}, false)
			transport := nativeTransport(t, f, protocol)
			image := uploadImage(t, transport, pixels)
			selected := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_SelectSession{SelectSession: &v1.SelectSession{Create: true}}}).GetState()
			if !selected.GetModel().GetImageInputSupported() {
				t.Fatal("vision capability missing from state")
			}
			command := &v1.Command{Operation: &v1.Command_Start{Start: &v1.StartRequest{SessionId: selected.GetSession().GetId(), Key: "attached-image-once", AttachmentIds: []string{image.ID}}}}
			reply := nativeCommand(t, transport, command)
			if reply.GetTask() == nil {
				t.Fatal(reply)
			}
			request := <-seen
			last := request.Messages[len(request.Messages)-1]
			if last.Role != "user" || len(last.Images) != 1 || !bytes.Equal(last.Images[0].Data, pixels) {
				t.Fatal("native image was lost")
			}
			waitNativeTask(t, deadline(t), f.app, reply.GetTask().GetId(), selected.Cursor)
			duplicate := nativeCommand(t, transport, command).GetTask()
			if duplicate.GetId() != reply.GetTask().GetId() {
				t.Fatal("image retry repeated task admission")
			}
			removed := nativeRequest(t, transport, "DELETE", "/v1/attachments/"+image.ID, "")
			if removed.Code != 204 {
				t.Fatal(removed.Code, removed.Body.String())
			}
			if _, err := f.app.ImageAttachments([]string{image.ID}); err == nil {
				t.Fatal("released upload still resolves")
			}
		})
	}
}
func TestVisionInputRejectsTextOnlyModel(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	transport := nativeTransport(t, f, "sse")
	image := uploadImage(t, transport, nativePNG(t))
	selected := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_SelectSession{SelectSession: &v1.SelectSession{Create: true}}}).GetState()
	reply := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_Start{Start: &v1.StartRequest{SessionId: selected.GetSession().GetId(), Key: "unsupported-image", AttachmentIds: []string{image.ID}}}})
	if reply.GetError().GetCode() != "failed_precondition" || len(f.app.State().Tasks) != 0 {
		t.Fatal("image sent to a text-only model", reply)
	}
}
