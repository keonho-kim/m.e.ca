//go:build linux && !android

// Purpose: Verify Wayland image transport, bounds and cancellation with real Unix sockets and descriptors.
// Pattern: Private compositor wire fixture.
// Usage: go test ./... inside this dependency on Linux; no desktop clipboard is accessed.
// Related: third_party/clipboard/image_wayland_linux.go, third_party/clipboard/image_wayland_wire_linux.go, third_party/clipboard/LICENSE.
package clipboard

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestImageWaylandDescriptorAndBounds(t *testing.T) {
	for _, size := range []int{128, 513} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			server := waylandImageFixture(t, func(file *os.File) { _, _ = file.Write(bytes.Repeat([]byte{'x'}, size)) })
			t.Setenv("WAYLAND_DISPLAY", server)
			value, err := ReadImage(ctx, 512)
			if size > 512 {
				if !errors.Is(err, ErrImageLimit) {
					t.Fatal(err)
				}
				return
			}
			if err != nil || len(value.Data) != size {
				t.Fatal(value, err)
			}
		})
	}
}

func TestImageWaylandCancelsOpenImagePipe(t *testing.T) {
	ready, release := make(chan struct{}), make(chan struct{})
	server := waylandImageFixture(t, func(*os.File) { close(ready); <-release })
	t.Setenv("WAYLAND_DISPLAY", server)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { _, err := ReadImage(ctx, 512); done <- err }()
	<-ready
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	close(release)
}

func TestImageWaylandCancelsUnresponsiveCompositor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wayland")
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	t.Setenv("WAYLAND_DISPLAY", path)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { _, err := ReadImage(ctx, 512); done <- err }()
	connection, err := listener.AcceptUnix()
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func waylandImageFixture(t *testing.T, write func(*os.File)) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "wayland")
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		c, err := listener.AcceptUnix()
		if err != nil {
			done <- err
			return
		}
		defer c.Close()
		_ = c.SetDeadline(time.Now().Add(5 * time.Second))
		for range 2 {
			if _, err = readWLRequest(c); err != nil {
				done <- err
				return
			}
		}
		global := func(id uint32, name string) []byte {
			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, id)
			b = append(b, wlEncodeString(name)...)
			return append(b, 1, 0, 0, 0)
		}
		for _, packet := range [][]byte{wlPacket(2, 0, global(10, "wl_seat")), wlPacket(2, 0, global(11, "ext_data_control_manager_v1")), wlPacket(3, 0, []byte{1, 0, 0, 0})} {
			if _, err = c.Write(packet); err != nil {
				done <- err
				return
			}
		}
		for range 4 {
			if _, err = readWLRequest(c); err != nil {
				done <- err
				return
			}
		}
		for _, packet := range [][]byte{wlPacket(6, 0, []byte{9, 0, 0, 0}), wlPacket(9, 0, wlEncodeString("image/png")), wlPacket(6, 1, []byte{9, 0, 0, 0}), wlPacket(7, 0, []byte{2, 0, 0, 0})} {
			if _, err = c.Write(packet); err != nil {
				done <- err
				return
			}
		}
		var data [512]byte
		var control [256]byte
		_, n, _, _, err := c.ReadMsgUnix(data[:], control[:])
		if err != nil {
			done <- err
			return
		}
		messages, err := syscall.ParseSocketControlMessage(control[:n])
		if err != nil {
			done <- err
			return
		}
		for _, message := range messages {
			fds, err := syscall.ParseUnixRights(&message)
			if err != nil {
				done <- err
				return
			}
			for _, fd := range fds {
				file := os.NewFile(uintptr(fd), "image-pipe")
				write(file)
				file.Close()
			}
		}
		done <- nil
	}()
	t.Cleanup(func() {
		listener.Close()
		if err := <-done; err != nil {
			t.Error(err)
		}
	})
	return path
}

func readWLRequest(r io.Reader) ([]byte, error) {
	var header [8]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}
	n := int(binary.LittleEndian.Uint32(header[4:]) >> 16)
	if n < 8 || n > 4096 {
		return nil, ErrImageLimit
	}
	body := make([]byte, n-8)
	_, err := io.ReadFull(r, body)
	return body, err
}
func wlPacket(object uint32, opcode uint16, body []byte) []byte {
	packet := make([]byte, 8+len(body))
	binary.LittleEndian.PutUint32(packet, object)
	binary.LittleEndian.PutUint32(packet[4:], uint32(len(packet))<<16|uint32(opcode))
	copy(packet[8:], body)
	return packet
}
