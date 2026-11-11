//go:build linux && !android

// Purpose: Receive bounded X11 image selections, including incremental transfers.
// Pattern: Request-owned selection transfer state machine.
// Usage: The Linux image selector calls this after Wayland is unavailable.
// Related: third_party/clipboard/image_x11_wire_linux.go, third_party/clipboard/image_linux.go, third_party/clipboard/LICENSE.
package clipboard

import (
	"context"
	"encoding/binary"
	"fmt"

	wire "golang.design/x/x11"
)

func imageX11(ctx context.Context, limit int) (Image, error) {
	x, err := openImageX11(ctx)
	if err != nil {
		return Image{}, err
	}
	defer x.close()
	selection, err := x.atom("CLIPBOARD")
	if err != nil {
		return Image{}, err
	}
	seq, err := x.send(wire.GetSelectionOwner(selection))
	if err != nil {
		return Image{}, err
	}
	owner, err := x.reply(seq, 65536)
	if err != nil {
		return Image{}, err
	}
	if owner.SelectionOwner() == 0 {
		return Image{}, ErrNoData
	}
	property, err := x.atom("MECA_IMAGE")
	if err != nil {
		return Image{}, err
	}
	targets, err := x.atom("TARGETS")
	if err != nil {
		return Image{}, err
	}
	available, err := x.selection(selection, targets, property, 65536)
	if err != nil {
		return Image{}, err
	}
	if len(available)%4 != 0 {
		return Image{}, ErrUnavailable
	}
	for _, mime := range imageTypes {
		target, err := x.atom(mime)
		if err != nil {
			return Image{}, err
		}
		found := false
		for i := 0; i < len(available); i += 4 {
			if binary.LittleEndian.Uint32(available[i:]) == target {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		budget := limit
		if mime == "text/uri-list" {
			budget = 8192
		}
		data, err := x.selection(selection, target, property, budget)
		if err != nil {
			return Image{}, err
		}
		return imageValue(mime, data)
	}
	return Image{}, ErrNoData
}

func (x *imageXConn) selection(selection, target, property uint32, limit int) ([]byte, error) {
	if _, err := x.send(wire.ConvertSelection(x.win, selection, target, property, wire.CurrentTime)); err != nil {
		return nil, err
	}
	for {
		p, err := x.packet(65536)
		if err != nil {
			return nil, err
		}
		if p.IsError() {
			return nil, ErrUnavailable
		}
		if p.IsEvent() && p.EventCode() == wire.EventSelectionNotify {
			n := p.SelectionNotify()
			if n.Requestor != x.win || n.Selection != selection || n.Target != target {
				continue
			}
			if n.Property == 0 {
				return nil, ErrNoData
			}
			break
		}
	}
	incr, err := x.atom("INCR")
	if err != nil {
		return nil, err
	}
	p, err := x.property(property, limit)
	if err != nil {
		return nil, err
	}
	if binary.LittleEndian.Uint32(p.Raw[8:]) != incr {
		return p.PropertyValue(), nil
	}
	hint := p.PropertyValue()
	// xclip legitimately sends an empty INCR size hint. The actual transfer,
	// not this advisory lower bound, remains constrained by limit.
	if len(hint) != 0 && len(hint) != 4 {
		return nil, fmt.Errorf("%w: INCR hint has %d bytes", ErrUnavailable, len(hint))
	}
	var estimated uint32
	if len(hint) == 4 {
		estimated = binary.LittleEndian.Uint32(hint)
	}
	if uint64(estimated) > uint64(limit) {
		return nil, ErrImageLimit
	}
	data := make([]byte, 0, int(estimated))
	for {
		p, err = x.packet(65536)
		if err != nil {
			return nil, err
		}
		if !p.IsEvent() || p.EventCode() != 28 || binary.LittleEndian.Uint32(p.Raw[4:]) != x.win || binary.LittleEndian.Uint32(p.Raw[8:]) != property || p.Raw[16] != 0 {
			continue
		}
		p, err = x.property(property, limit-len(data))
		if err != nil {
			return nil, err
		}
		part := p.PropertyValue()
		if len(part) == 0 {
			return data, nil
		}
		needed := len(data) + len(part)
		if needed > limit {
			return nil, ErrImageLimit
		}
		if cap(data) < needed {
			grown := make([]byte, len(data), min(limit, max(needed, max(65536, cap(data)*2))))
			copy(grown, data)
			data = grown
		}
		data = append(data, part...)
	}
}

func (x *imageXConn) property(atom uint32, limit int) (wire.Packet, error) {
	seq, err := x.send(wire.GetProperty(true, x.win, atom, 0, 0, uint32((limit+3)/4)))
	if err != nil {
		return wire.Packet{}, err
	}
	p, err := x.reply(seq, limit+4)
	if err == nil && (binary.LittleEndian.Uint32(p.Raw[12:]) > 0 || len(p.PropertyValue()) > limit) {
		return wire.Packet{}, ErrImageLimit
	}
	return p, err
}
