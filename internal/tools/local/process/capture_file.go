// Purpose: Keep raw command output private while spilling bounded captures to disk.
// Pattern: Encrypted fixed-block file with one resident block and an ephemeral key.
// Usage: spillCapture owns the file; release closes and removes it after archival.
// Related: internal/tools/local/process/spill.go, internal/tools/local/process/output.go.
package process

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"os"
)

const captureBlock = 64 * 1024

type captureFile struct {
	file    *os.File
	aead    cipher.AEAD
	plain   []byte
	encoded []byte
	known   [2 * captureHalf / captureBlock]bool
	block   int
	dirty   bool
}

func newCaptureFile(directory string) (*captureFile, error) {
	var key [32]byte
	if _, err := rand.Read(key[:]); err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key[:])
	clear(key[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	file, err := os.CreateTemp(directory, "capture-*.part")
	if err != nil {
		return nil, err
	}
	return &captureFile{file: file, aead: aead, block: -1,
		plain: make([]byte, captureBlock), encoded: make([]byte, captureBlock+aead.NonceSize()+aead.Overhead())}, nil
}

func (f *captureFile) flush() error {
	if !f.dirty {
		return nil
	}
	nonce := f.encoded[:f.aead.NonceSize()]
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	f.encoded = f.aead.Seal(nonce, nonce, f.plain, nil)
	if _, err := f.file.WriteAt(f.encoded, int64(f.block*len(f.encoded))); err != nil {
		return err
	}
	f.known[f.block], f.dirty = true, false
	return nil
}

func (f *captureFile) load(block int) error {
	if block == f.block {
		return nil
	}
	if block < 0 || block >= len(f.known) {
		return io.EOF
	}
	if err := f.flush(); err != nil {
		return err
	}
	f.block = -1
	clear(f.plain)
	if f.known[block] {
		if _, err := f.file.ReadAt(f.encoded, int64(block*len(f.encoded))); err != nil {
			return err
		}
		nonce := f.encoded[:f.aead.NonceSize()]
		if _, err := f.aead.Open(f.plain[:0], nonce, f.encoded[len(nonce):], nil); err != nil {
			return err
		}
	}
	f.block = block
	return nil
}

func (f *captureFile) WriteAt(value []byte, offset int64) (int, error) {
	if offset < 0 {
		return 0, errors.New("negative capture offset")
	}
	written := 0
	for len(value) > 0 {
		if err := f.load(int(offset / captureBlock)); err != nil {
			return written, err
		}
		n := copy(f.plain[offset%captureBlock:], value)
		f.dirty = true
		written, offset, value = written+n, offset+int64(n), value[n:]
	}
	return written, nil
}

func (f *captureFile) ReadAt(value []byte, offset int64) (int, error) {
	if offset < 0 {
		return 0, errors.New("negative capture offset")
	}
	read := 0
	for len(value) > 0 {
		if err := f.load(int(offset / captureBlock)); err != nil {
			return read, err
		}
		n := copy(value, f.plain[offset%captureBlock:])
		read, offset, value = read+n, offset+int64(n), value[n:]
	}
	return read, nil
}

func (f *captureFile) Name() string { return f.file.Name() }

func (f *captureFile) Close() error {
	clear(f.plain)
	clear(f.encoded)
	f.plain, f.encoded, f.aead = nil, nil, nil
	return f.file.Close()
}
