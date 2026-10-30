// Purpose: Verify image snapshots obey filesystem policy and do not count as file changes.
// Pattern: Filesystem integration tests.
// Usage: go test ./internal/tools/local -run TestReadImage.
// Related: internal/tools/local/image.go, internal/tools/local/prepare.go.
package local

import (
	"bytes"
	"context"
	"github.com/keonho-kim/meca/internal/policy"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadImageSnapshotAndPathPolicy(t *testing.T) {
	executor, root := testExecutor(t)
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewGray(image.Rect(0, 0, 12, 10))); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "shot.bin")
	if err := os.WriteFile(path, data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	op, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.Directory}, call("read_image", `{"path":"shot.bin"}`))
	if err != nil {
		t.Fatal(err)
	}
	if op.Approval().Required || op.Approval().Diff {
		t.Fatal("read requested mutation approval")
	}
	result, err := op.Execute(context.Background(), true)
	if err != nil || result.Started || len(result.Changes) > 0 || len(result.Images) != 1 || result.Images[0].Width != 12 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if len(result.Content) > 1024 || strings.Contains(string(result.Content), "data:image") {
		t.Fatal("binary leaked to text output")
	}
	before := append([]byte(nil), result.Images[0].Data...)
	if err := os.WriteFile(path, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result.Images[0].Data, before) {
		t.Fatal("snapshot changed with its source file")
	}
	for _, path := range []string{".", "../outside"} {
		op, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.Directory}, call("read_image", `{"path":"`+path+`"}`))
		if err == nil {
			_, err = op.Execute(context.Background(), true)
		}
		if err == nil {
			t.Fatalf("invalid target accepted: %s", path)
		}
	}
}
