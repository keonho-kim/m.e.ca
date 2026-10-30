// Purpose: Exercise dynamic image tools, observation context and read-only accounting through one main loop.
// Pattern: Scripted-model integration test over real file and SQLite adapters.
// Usage: go test ./internal/agent/session -run TestImage.
// Related: internal/commands/agentrequest/request.go, internal/tools/local/image.go.
package session_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/session"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/commands/agentrequest"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"github.com/keonho-kim/meca/internal/tools/local"
)

func TestImageMainLoopAndTextModelSwitch(t *testing.T) {
	for _, reject := range []bool{false, true} {
		t.Run(fmt.Sprint("reject=", reject), func(t *testing.T) { runImageSession(t, reject) })
	}
}

func runImageSession(t *testing.T, reject bool) {
	ctx, project := context.Background(), t.TempDir()
	var imageBytes bytes.Buffer
	if err := png.Encode(&imageBytes, image.NewGray(image.Rect(0, 0, 10, 12))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "shot.png"), imageBytes.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	sessions, err := sqlite.OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := sessions.Create(ctx, project, project)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	tools, err := local.NewExecutor(project, t.TempDir(), t.TempDir(), func(s string) string { return s })
	if err != nil {
		t.Fatal(err)
	}
	defer tools.Close()
	selected := connection.Selection{Model: "vision", Connection: connection.Connection{ID: "provider"}, Details: &connection.ModelInfo{ID: "vision", ImageInput: connection.DeclaredImageInput(true, "test")}}
	step := 0
	generator := generation(func(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
		step++
		imageTool := false
		for _, d := range request.Tools {
			imageTool = imageTool || d.Name == "read_image"
		}
		if imageTool != (request.ModelID == "vision" && (!reject || step < 3)) {
			t.Fatal("incorrect dynamic image tool selection")
		}
		if step == 1 {
			return emit(modelapi.Event{Done: true, Calls: []toolapi.Call{{ID: "read", Name: "read_image", Arguments: json.RawMessage(`{"path":"shot.png"}`)}}})
		}
		found := false
		for _, m := range request.Messages {
			found = found || len(m.Images) > 0
		}
		if found != (request.ModelID == "vision" && (!reject || step < 3)) {
			t.Fatal("image context lost or sent to text model")
		}
		if reject && step == 2 {
			return &modelapi.ImageInputError{}
		}
		return emit(modelapi.Event{Done: true, Delta: "Completed"})
	})
	for _, model := range []string{"vision", "text"} {
		selected.Model = model
		selected.Details.ID = model
		selected.Details.ImageInput = connection.DeclaredImageInput(model == "vision", "test")
		cfg := config.Defaults()
		req := agentrequest.New(agentrequest.Options{Config: cfg, Selection: selected, Session: store.Session(), Generator: generator, Tools: tools, Store: store, Vault: &credentials.Vault{State: t.TempDir()}, Text: "Inspect image", LoadPrompts: func() (prompt.Templates, error) { return prompt.Defaults(), nil }})
		_, err := session.Execute(ctx, req, func(context.Context, toolapi.Approval) (bool, error) {
			t.Fatal("read requested approval")
			return false, nil
		}, func(event runrecord.Event) error {
			if event.Checkpoint != nil && event.Checkpoint.ChangeRevision != 0 {
				t.Fatal("image read counted as mutation")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	expected := 3
	if reject {
		expected = 4
	}
	if step != expected {
		t.Fatalf("unexpected loop count: %d", step)
	}
}
