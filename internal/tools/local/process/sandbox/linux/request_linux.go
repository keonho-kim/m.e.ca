//go:build linux

// Purpose: Transfer a bounded immutable launch request to the internal child through an inherited descriptor.
// Pattern: Private process protocol.
// Usage: Prepare writes descriptor 3; the child validates it before touching namespaces.
// Related: internal/tools/local/process/sandbox/linux.go, internal/tools/local/process/sandbox/linux/child_linux.go.
package linux

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

const ChildName = "v-internal-project-shell"
const maxRequest = 2 << 20

type Request struct {
	Project, Private, Scratch, Command string
	Runtime                            []string
	UserRuntime                        []string
}

func (r Request) Validate() error {
	for _, path := range append(append([]string{r.Project, r.Private, r.Scratch}, r.Runtime...), r.UserRuntime...) {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsRune(path, 0) {
			return fmt.Errorf("invalid sandbox path: %q", path)
		}
	}
	if r.Command == "" || strings.ContainsRune(r.Command, 0) || len(r.Command) > maxRequest/2 {
		return fmt.Errorf("invalid sandbox command")
	}
	if within(r.Private, r.Project) || within(r.Project, r.Scratch) || within(r.Scratch, r.Project) {
		return fmt.Errorf("project, V state and temporary sandbox paths overlap unsafely")
	}
	for _, path := range append(append([]string{}, r.Runtime...), "/proc", "/dev") {
		if within(path, r.Project) || within(r.Project, path) {
			return fmt.Errorf("project overlaps a system runtime path: %s", path)
		}
	}
	for _, path := range r.UserRuntime {
		if within(path, r.Project) || within(path, r.Private) || within(r.Private, path) || within(path, r.Scratch) || within(r.Scratch, path) {
			return fmt.Errorf("user runtime overlaps protected or working paths: %s", path)
		}
	}
	return nil
}

func within(parent, path string) bool {
	relative, err := filepath.Rel(parent, path)
	return err == nil && (relative == "." || filepath.IsLocal(relative))
}

func Encode(request Request) (_ *os.File, err error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	if len(encoded) > maxRequest {
		return nil, fmt.Errorf("sandbox request exceeds %d bytes", maxRequest)
	}
	fd, err := unix.MemfdCreate("v-shell-request", unix.MFD_CLOEXEC|unix.MFD_ALLOW_SEALING)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), "v-shell-request")
	defer func() {
		if err != nil {
			err = errors.Join(err, file.Close())
		}
	}()
	if _, err = file.Write(encoded); err != nil {
		return nil, err
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	_, err = unix.FcntlInt(file.Fd(), unix.F_ADD_SEALS, unix.F_SEAL_WRITE|unix.F_SEAL_GROW|unix.F_SEAL_SHRINK|unix.F_SEAL_SEAL)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func readRequest() (request Request, err error) {
	file := os.NewFile(3, "v-shell-request")
	defer func() { err = errors.Join(err, file.Close()) }()
	seals, err := unix.FcntlInt(file.Fd(), unix.F_GET_SEALS, 0)
	if err != nil || seals&unix.F_SEAL_WRITE == 0 {
		return request, fmt.Errorf("missing immutable internal launch request")
	}
	bytes, err := io.ReadAll(io.LimitReader(file, maxRequest+1))
	if err != nil {
		return request, err
	}
	if len(bytes) > maxRequest {
		return request, fmt.Errorf("sandbox request too large")
	}
	if err := json.Unmarshal(bytes, &request); err != nil {
		return request, err
	}
	return request, request.Validate()
}
