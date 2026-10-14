//go:build linux

// Purpose: Reject invalid or overlapping launch roots and preserve literal commands in immutable requests.
// Pattern: Process protocol contract test.
// Usage: go test ./internal/tools/local/process/sandbox/linux.
// Related: internal/tools/local/process/sandbox/linux/request_linux.go.
package linux

import (
	"encoding/json"
	"errors"
	"testing"

	"golang.org/x/sys/unix"
)

func TestRequestBoundaries(t *testing.T) {
	valid := Request{Project: "/home/user/project", Private: "/home/user/.v", Scratch: "/tmp/v-command", Runtime: []string{"/bin", "/usr/bin"}, Command: "printf '%s' '$(literal)'"}
	for _, project := range []string{".", "/", "/usr/bin/work", "/home/user/.v/work", "/tmp", "/tmp/v-command/nested"} {
		request := valid
		request.Project = project
		if err := request.Validate(); err == nil {
			t.Fatalf("accepted %s", project)
		}
	}
	valid.Private = valid.Project + "/.v"
	file, err := Encode(valid)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	}()
	var decoded Request
	if err := json.NewDecoder(file).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Command != valid.Command {
		t.Fatalf("changed literal command: %q", decoded.Command)
	}
	if _, err := file.WriteString("changed"); !errors.Is(err, unix.EPERM) {
		t.Fatalf("request write was not denied: %v", err)
	}
}
