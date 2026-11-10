// Purpose: Verify WebUI model and file operations target explicit sessions without cross-tab selection changes.
// Pattern: Native transport contract integration tests.
// Usage: go test -race ./internal/serverapi -run TestWorkspace.
// Related: internal/serverapi/workspace.go, internal/commands/server/target.go.
package serverapi

import (
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func TestWorkspaceSessionIsolationAndDiscovery(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	transport := nativeTransport(t, f, "sse")
	first := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_SelectSession{SelectSession: &v1.SelectSession{Create: true}}}).GetState().GetSession().GetId()
	second := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_SelectSession{SelectSession: &v1.SelectSession{Create: true}}}).GetState().GetSession().GetId()
	if err := os.WriteFile(filepath.Join(f.project, "notes.txt"), []byte("workspace preview"), 0600); err != nil {
		t.Fatal(err)
	}
	selection := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_SelectFile{SelectFile: &v1.FileSelection{SessionId: first, Path: "notes.txt", Selected: true}}})
	if selection.GetError() != nil {
		t.Fatal(selection)
	}
	paths := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_FileSelections{FileSelections: &v1.SessionTarget{SessionId: first}}}).GetFileSelections().GetPaths()
	if len(paths) != 1 || paths[0] != "notes.txt" || f.app.State().Session.ID != second {
		t.Fatal("reference selection changed another tab", paths)
	}
	directory := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_Directory{Directory: &v1.FileRequest{Path: ".", Limit: 10}}})
	if len(directory.GetDirectory().GetItems()) != 1 {
		t.Fatal(directory)
	}
	preview := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_PreviewFile{PreviewFile: &v1.FileRequest{Path: "notes.txt", Limit: 9}}}).GetFilePreview()
	if preview.GetText() != "workspace" || !preview.GetTruncated() || preview.GetNextOffset() != 9 {
		t.Fatal(preview)
	}
	continued := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_PreviewFile{PreviewFile: &v1.FileRequest{Path: "notes.txt", Offset: 9, Limit: 64 << 10, Revision: preview.GetRevision()}}}).GetFilePreview()
	if continued.GetText() != " preview" {
		t.Fatal(continued)
	}
	providers := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_Providers{Providers: &v1.Empty{}}}).GetProviders()
	if len(providers.GetItems()) != 9 {
		t.Fatal(providers)
	}
	models := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_DiscoverModels{DiscoverModels: &v1.ModelQuery{SessionId: first, Provider: "ollama"}}})
	if len(models.GetModels().GetItems()) != 1 || f.app.State().Session.ID != second {
		t.Fatal(models)
	}
	missingTarget := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_SetModel{SetModel: &v1.ModelRequest{Provider: "ollama", Model: "fixture"}}})
	if missingTarget.GetError().GetCode() != "invalid_argument" {
		t.Fatal("ambiguous model mutation accepted", missingTarget)
	}
	model := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_SetModel{SetModel: &v1.ModelRequest{SessionId: first, Provider: "ollama", Model: "fixture"}}})
	if model.GetError() != nil || model.GetState().GetSession().GetId() != first || f.app.State().Session.ID != second {
		t.Fatal(model)
	}
	language := "ko"
	updated := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_SetSettings{SetSettings: &v1.SettingsRequest{Language: &language}}})
	if updated.GetState().GetLanguage() != language {
		t.Fatal(updated)
	}
	caps := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_Capabilities{Capabilities: &v1.Empty{}}}).GetCapabilities()
	if caps.GetInputImages() != 8 || caps.GetImageBytes() != 64<<20 || caps.GetApiVersion() != "meca.v1" {
		t.Fatal(caps)
	}
}
