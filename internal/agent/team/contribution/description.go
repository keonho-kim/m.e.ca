// Purpose: Bound explanatory state while keeping full-content equality independent of truncation.
// Pattern: Pure input projection before shared-state locks.
// Usage: SetWork describes incoming text before preparing or publishing its candidate.
// Related: internal/agent/team/contribution/state.go, internal/agent/team/work.go.
package contribution

import (
	"crypto/sha256"
	"hash"
	"unicode/utf8"
)

const summaryContextBytes = 4 * 1024

// Describe hashes original text once and retains only self-contained previews.
func Describe(value Scope) Scope {
	if len(value.Summary) <= summaryContextBytes && len(value.Contract) <= summaryContextBytes {
		value.summaryHash = smallFingerprint(value.Summary)
		value.contractHash = smallFingerprint(value.Contract)
	} else {
		// Hashing large accepted input never allocates a byte-for-byte source copy.
		scratch := make([]byte, 16*1024)
		h := sha256.New()
		value.summaryHash = textFingerprint(value.Summary, scratch, h)
		value.contractHash = textFingerprint(value.Contract, scratch, h)
	}
	var joined [64]byte
	member := smallFingerprint(value.Member)
	copy(joined[:32], member[:])
	copy(joined[32:], value.contractHash[:])
	value.progressHash = sha256.Sum256(joined[:])
	for _, path := range value.Paths {
		var key [32]byte
		if len(path) <= summaryContextBytes {
			key = smallFingerprint(path)
		} else {
			key = textFingerprint(path, make([]byte, 4096), sha256.New())
		}
		copy(joined[:32], value.progressHash[:])
		copy(joined[32:], key[:])
		value.progressHash = sha256.Sum256(joined[:])
	}
	value.Summary = summaryPreview(value.Summary)
	value.Contract = summaryPreview(value.Contract)
	return value
}

func smallFingerprint(text string) [32]byte {
	var scratch [summaryContextBytes]byte
	n := copy(scratch[:], text)
	return sha256.Sum256(scratch[:n])
}

func textFingerprint(text string, scratch []byte, h hash.Hash) [32]byte {
	h.Reset()
	for len(text) > 0 {
		n := copy(scratch, text)
		_, _ = h.Write(scratch[:n]) // The standard SHA-256 writer cannot fail.
		text = text[n:]
	}
	var out [32]byte
	h.Sum(out[:0])
	return out
}

func summaryPreview(text string) string {
	if len(text) <= summaryContextBytes {
		return text
	}
	const suffix = "… (full declaration in public conversation)"
	end := summaryContextBytes - len(suffix)
	for !utf8.RuneStart(text[end]) {
		end--
	}
	return text[:end] + suffix
}
