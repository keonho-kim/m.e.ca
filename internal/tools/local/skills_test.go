// Purpose: Verify native skill reads preserve private-path guards and reuse bounded image decoding.
// Pattern: Local tool integration tests.
// Usage: go test ./internal/tools/local -run TestSkill.
// Related: internal/tools/local/skills.go, internal/tools/local/skill_image.go.
package local

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/policy"
)

func TestSkillImageUsesExistingDecoderOnlyForVision(t *testing.T) {
	e, project := testExecutor(t)
	dir := filepath.Join(project, ".agents/skills/picture")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: picture\ndescription: Read picture\n---\nUse image.png"), 0600); err != nil {
		t.Fatal(err)
	}
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewGray(image.Rect(0, 0, 12, 10))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "image.png"), data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	for _, vision := range []bool{false, true} {
		e.SetSkillImageInput(vision)
		op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.Directory}, call("read_skill_file", `{"id":"project:picture","path":"image.png"}`))
		if err != nil {
			t.Fatal(err)
		}
		result, err := op.Execute(context.Background(), true)
		if err != nil {
			t.Fatal(err)
		}
		if vision && (len(result.Images) != 1 || result.Images[0].Width != 12) || !vision && len(result.Images) != 0 {
			t.Fatalf("vision=%v images=%d", vision, len(result.Images))
		}
	}
}

func TestSkillPathHonorsDirectoryScope(t *testing.T) {
	e, _ := testExecutor(t)
	dir := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: outside\ndescription: Outside source\n---\nInstructions"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, scope := range []policy.Scope{policy.Directory, policy.System} {
		body, err := json.Marshal(map[string]string{"path": dir})
		if err != nil {
			t.Fatal(err)
		}
		op, err := e.Prepare(context.Background(), policy.Settings{Scope: scope}, call("load_skill", string(body)))
		if err == nil {
			_, err = op.Execute(context.Background(), true)
		}
		if scope == policy.Directory && err == nil || scope == policy.System && err != nil {
			t.Fatalf("scope=%v err=%v", scope, err)
		}
	}
}
