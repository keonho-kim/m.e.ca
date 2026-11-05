// Purpose: Quantize resident bytes into comparable display values before allocating header text.
// Pattern: Pure presentation value.
// Usage: Session compares MemorySize snapshots and formats only changed display values.
// Related: internal/tui/session/memory_usage.go, internal/tui/view/tokens.go.
package view

import "strconv"

// MemorySize is a comparable RSS display value, not an additional raw sample.
type MemorySize struct {
	amount uint64
	scale  uint8
}

// ResidentMemory rounds RSS to tenths of MiB or hundredths of GiB using integer arithmetic.
func ResidentMemory(bytes uint64) MemorySize {
	if bytes == 0 {
		return MemorySize{}
	}
	unit, scale := uint64(1<<20), uint64(10)
	if bytes >= 1<<30 {
		unit, scale = 1<<30, 100
	}
	return MemorySize{amount: bytes/unit*scale + (bytes%unit*scale+unit/2)/unit, scale: uint8(scale)}
}

// Label allocates one header string for an available value and reuses the unavailable literal.
func (m MemorySize) Label() string {
	if m.scale == 0 {
		return "MEM. USAGE —"
	}
	var buffer [64]byte
	line := append(buffer[:0], "MEM. USAGE "...)
	scale := uint64(m.scale)
	line = strconv.AppendUint(line, m.amount/scale, 10)
	line = append(line, '.')
	if scale == 100 && m.amount%scale < 10 {
		line = append(line, '0')
	}
	line = strconv.AppendUint(line, m.amount%scale, 10)
	unit := " MiB"
	if scale == 100 {
		unit = " GiB"
	}
	return string(append(line, unit...))
}
