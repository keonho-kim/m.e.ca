//go:build linux && !android

// Purpose: Own Wayland socket cancellation and bound control packets and ancillary descriptors.
// Pattern: Bounded extension of the upstream Wayland wire connection.
// Usage: Image reads use this connection instead of global package initialization.
// Related: third_party/clipboard/image_wayland_linux.go, third_party/clipboard/clipboard_wayland.go, third_party/clipboard/LICENSE.
package clipboard

import (
	"context"
	"net"
	"syscall"
)

type imageWLConn struct {
	*wlConn
	stop     func() bool
	metadata int
}

func openImageWayland(ctx context.Context) (*imageWLConn, error) {
	path := waylandSocketPath()
	if path == "" {
		return nil, ErrUnavailable
	}
	c, err := (&net.Dialer{}).DialContext(ctx, "unix", path)
	if err != nil {
		return nil, ErrUnavailable
	}
	u := c.(*net.UnixConn)
	if deadline, ok := ctx.Deadline(); ok {
		_ = u.SetDeadline(deadline)
	}
	return &imageWLConn{wlConn: &wlConn{c: u, nextID: wlDisplayID + 1}, stop: context.AfterFunc(ctx, func() { _ = u.Close() })}, nil
}
func (w *imageWLConn) close() {
	w.stop()
	_ = w.c.Close()
	for _, fd := range w.fds {
		_ = syscall.Close(fd)
	}
	w.fds = nil
}
func (w *imageWLConn) next() (uint32, uint16, []byte, error) {
	obj, op, body, err := w.readEvent()
	w.metadata += len(body) + 8
	if len(w.fds) > 0 || w.metadata > 65536 {
		return 0, 0, nil, ErrImageLimit
	}
	return obj, op, body, err
}
