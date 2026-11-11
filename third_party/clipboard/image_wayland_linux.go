//go:build linux && !android

// Purpose: Read native Wayland image offers with bounded metadata, data and cancellation.
// Pattern: Request-scoped data-control state machine.
// Usage: The Linux image adapter prefers this when a data-control manager is available.
// Related: third_party/clipboard/image_wayland_wire_linux.go, third_party/clipboard/image_linux.go, third_party/clipboard/LICENSE.
package clipboard

import (
	"context"
	"encoding/binary"
	"io"
	"os"
)

func imageWayland(ctx context.Context, limit int) (Image, error) {
	w, err := openImageWayland(ctx)
	if err != nil {
		return Image{}, err
	}
	defer w.close()
	registry := w.newID()
	arg := make([]byte, 4)
	binary.LittleEndian.PutUint32(arg, registry)
	if err = w.request(wlDisplayID, dispOpcodeGetRegistry, arg); err != nil {
		return Image{}, err
	}
	first, err := w.sync()
	if err != nil {
		return Image{}, err
	}
	globals := map[string]wlGlobal{}
	for {
		obj, op, body, err := w.next()
		if err != nil {
			return Image{}, err
		}
		if obj == wlDisplayID && op == 0 {
			return Image{}, wlDisplayError(body)
		}
		if obj == registry && op == 0 && len(body) >= 4 {
			name, offset, e := wlString(body, 4)
			if e != nil || offset+4 > len(body) {
				return Image{}, ErrUnavailable
			}
			if name == "wl_seat" || name == dataControlManagers[0] || name == dataControlManagers[1] {
				globals[name] = wlGlobal{name: binary.LittleEndian.Uint32(body), version: binary.LittleEndian.Uint32(body[offset:])}
			}
		}
		if obj == first && op == 0 {
			break
		}
	}
	iface, manager, ok := dataControlManager(globals)
	if !ok {
		return Image{}, ErrUnavailable
	}
	seat, ok := globals["wl_seat"]
	if !ok {
		return Image{}, ErrUnavailable
	}
	version, _ := dataControlBind(iface, manager.version)
	managerID, err := w.bind(registry, manager.name, iface, version)
	if err != nil {
		return Image{}, err
	}
	seatID, err := w.bind(registry, seat.name, "wl_seat", 1)
	if err != nil {
		return Image{}, err
	}
	device := w.newID()
	arg = make([]byte, 8)
	binary.LittleEndian.PutUint32(arg, device)
	binary.LittleEndian.PutUint32(arg[4:], seatID)
	if err = w.request(managerID, mgrOpcodeGetDataDevice, arg); err != nil {
		return Image{}, err
	}
	second, err := w.sync()
	if err != nil {
		return Image{}, err
	}
	offers := map[uint32][]string{}
	var current uint32
	for {
		obj, op, body, err := w.next()
		if err != nil {
			return Image{}, err
		}
		if obj == wlDisplayID && op == 0 {
			return Image{}, wlDisplayError(body)
		}
		switch {
		case obj == device && op == devEvtDataOffer && len(body) == 4:
			if len(offers) >= 128 {
				return Image{}, ErrImageLimit
			}
			offers[binary.LittleEndian.Uint32(body)] = nil
		case obj == device && op == devEvtSelection && len(body) == 4:
			current = binary.LittleEndian.Uint32(body)
		case op == offerEvtOffer:
			if _, exists := offers[obj]; exists {
				mime, _, err := wlString(body, 0)
				if err != nil {
					return Image{}, err
				}
				for _, allowed := range imageTypes {
					if mime == allowed {
						offers[obj] = append(offers[obj], mime)
						break
					}
				}
			}
		}
		if obj == second && op == 0 {
			break
		}
	}
	if current == 0 {
		return Image{}, ErrNoData
	}
	mime := pickMIME(imageTypes, offers[current])
	if mime == "" {
		return Image{}, ErrNoData
	}
	if mime == "text/uri-list" {
		limit = 8192
	}
	data, err := w.receive(ctx, current, mime, limit)
	if err != nil {
		return Image{}, err
	}
	return imageValue(mime, data)
}

func (w *imageWLConn) receive(ctx context.Context, offer uint32, mime string, limit int) ([]byte, error) {
	r, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	if err = w.requestFd(offer, offerOpcodeReceive, wlEncodeString(mime), int(writer.Fd())); err != nil {
		writer.Close()
		return nil, err
	}
	if err = writer.Close(); err != nil {
		return nil, err
	}
	stop := context.AfterFunc(ctx, func() { _ = r.Close() })
	defer stop()
	data, err := io.ReadAll(io.LimitReader(r, int64(limit)+1))
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if len(data) > limit {
		return nil, ErrImageLimit
	}
	return data, err
}
