//go:build linux && !android

// Purpose: Own cancellable X11 connections and validate reply lengths before allocation.
// Pattern: Bounded wire adapter using the upstream protocol codecs.
// Usage: Image selection transfers use one connection and an unshown requestor window.
// Related: third_party/clipboard/image_x11_linux.go, third_party/clipboard/LICENSE.
package clipboard

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	wire "golang.design/x/x11"
)

type imageXConn struct {
	c    net.Conn
	r    *bufio.Reader
	win  uint32
	seq  uint16
	stop func() bool
}

func openImageX11(ctx context.Context) (*imageXConn, error) {
	d, err := wire.ParseDisplay(os.Getenv("DISPLAY"))
	if err != nil {
		return nil, ErrUnavailable
	}
	c, err := (&net.Dialer{}).DialContext(ctx, d.Net, d.Addr)
	if err != nil && d.Net == "unix" {
		c, err = (&net.Dialer{}).DialContext(ctx, "unix", "@/tmp/.X11-unix/X"+strconv.Itoa(d.Num))
	}
	if err != nil {
		return nil, ErrUnavailable
	}
	x := &imageXConn{c: c, r: bufio.NewReader(c), stop: context.AfterFunc(ctx, func() { _ = c.Close() })}
	ok := false
	defer func() {
		if !ok {
			x.close()
		}
	}()
	if deadline, exists := ctx.Deadline(); exists {
		_ = c.SetDeadline(deadline)
	}
	name, data := imageCookie(d.Num)
	if _, err = c.Write(wire.SetupRequest(name, data)); err != nil {
		return nil, err
	}
	header, err := x.r.Peek(8)
	if err != nil {
		return nil, err
	}
	if int(binary.LittleEndian.Uint16(header[6:]))*4 > 65536 {
		return nil, ErrImageLimit
	}
	setup, err := wire.ReadSetup(x.r)
	if err != nil {
		return nil, ErrUnavailable
	}
	x.win = wire.NewIDGen(setup).Next()
	if _, err = x.send(wire.CreateWindow(x.win, setup.Root)); err != nil {
		return nil, err
	}
	// CWEventMask / PropertyChangeMask lets us receive INCR chunks.
	attributes := make([]byte, 16)
	attributes[0] = 2
	binary.LittleEndian.PutUint16(attributes[2:], 4)
	binary.LittleEndian.PutUint32(attributes[4:], x.win)
	binary.LittleEndian.PutUint32(attributes[8:], 1<<11)
	binary.LittleEndian.PutUint32(attributes[12:], 1<<22)
	if _, err = x.send(attributes); err != nil {
		return nil, err
	}
	seq, err := x.send(wire.GetInputFocus())
	if err != nil {
		return nil, err
	}
	if _, err = x.reply(seq, 65536); err != nil {
		return nil, err
	}
	ok = true
	return x, nil
}

func imageCookie(display int) (string, []byte) {
	path := os.Getenv("XAUTHORITY")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", nil
		}
		path = filepath.Join(home, ".Xauthority")
	}
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", nil
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 65536 {
		return "", nil
	}
	data, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil || len(data) > 65536 {
		return "", nil
	}
	entries, err := wire.ParseXauthority(data)
	if err != nil {
		return "", nil
	}
	host, _ := os.Hostname()
	return wire.ChooseCookie(entries, display, host)
}

func (x *imageXConn) close() { x.stop(); _ = x.c.Close() }
func (x *imageXConn) send(data []byte) (uint16, error) {
	_, err := x.c.Write(data)
	x.seq++
	return x.seq, err
}
func (x *imageXConn) packet(limit int) (wire.Packet, error) {
	h, err := x.r.Peek(32)
	if err != nil {
		return wire.Packet{}, err
	}
	if h[0] == 1 && uint64(binary.LittleEndian.Uint32(h[4:]))*4 > uint64(limit) {
		return wire.Packet{}, ErrImageLimit
	}
	return wire.ReadPacket(x.r)
}
func (x *imageXConn) reply(seq uint16, limit int) (wire.Packet, error) {
	for {
		p, err := x.packet(limit)
		if err != nil {
			return p, err
		}
		if p.IsError() {
			return p, fmt.Errorf("%w: X11 error %d at sequence %d (waiting for %d)", ErrUnavailable, p.ErrorCode(), p.Sequence(), seq)
		}
		if p.IsReply() && p.Sequence() == seq {
			return p, nil
		}
	}
}
func (x *imageXConn) atom(name string) (uint32, error) {
	seq, err := x.send(wire.InternAtom(name, false))
	if err != nil {
		return 0, err
	}
	p, err := x.reply(seq, 65536)
	if err != nil {
		return 0, err
	}
	return p.Atom(), nil
}
