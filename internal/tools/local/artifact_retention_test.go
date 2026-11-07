// Purpose: Bound retained command logs while preserving every result not yet acknowledged as durable.
// Pattern: Controlled filesystem quota and immutable-result tests.
// Usage: go test -race ./internal/tools/local -run TestArtifact.
// Related: internal/tools/local/artifact_retention.go, internal/tools/local/command_archive.go.
package local

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/storage/capacity"
)

func TestArtifactRetentionReclaimsOnlyAcknowledgedResults(t *testing.T) {
	e, _ := testExecutor(t)
	e.retained.limitBytes = 100
	e.retained.limitCommands = 2
	body := []byte(strings.Repeat("x", 40))
	path := func(id string) string {
		p, err := e.commandPath(id, ".log")
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	for _, id := range []string{"cmd_first", "cmd_pending"} {
		if err := e.writeArtifact(id, path(id), body); err != nil {
			t.Fatal(err)
		}
	}
	if err := e.writeArtifact("cmd_next", path("cmd_next"), body); !errors.Is(err, capacity.ErrLimit) {
		t.Fatal("unacknowledged result discarded", err)
	}
	if err := e.AcknowledgeCommand("cmd_first"); err != nil {
		t.Fatal(err)
	}
	if err := e.writeArtifact("cmd_next", path("cmd_next"), body); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path("cmd_first")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("expired temporary log remained", err)
	}
	if got, err := os.ReadFile(path("cmd_pending")); err != nil || string(got) != string(body) {
		t.Fatal("pending original lost", err)
	}
	if e.retained.bytes > 100 || len(e.retained.entries) > 2 {
		t.Fatal("retention budget exceeded")
	}
}

func TestArtifactReplacementRemainsBoundedAndCompletedIsImmutable(t *testing.T) {
	e, _ := testExecutor(t)
	e.retained.limitBytes = 64
	path, _ := e.commandPath("cmd_owned", ".log")
	if err := e.writeArtifact("cmd_owned", path, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := e.writeArtifact("cmd_owned", path, []byte("replacement")); err != nil {
		t.Fatal(err)
	}
	if err := e.writeArtifact("cmd_owned", path, make([]byte, 65)); !errors.Is(err, capacity.ErrLimit) {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != "replacement" {
		t.Fatal("failed replacement changed the file", err)
	}
	if err := e.AcknowledgeCommand("cmd_owned"); err != nil {
		t.Fatal(err)
	}
	if err := e.writeArtifact("cmd_owned", path, []byte("late")); err == nil {
		t.Fatal("completed archive was overwritten")
	}
}
