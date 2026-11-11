// Purpose: Reuse validated gzip payloads and publish build files without partial reads.
// Pattern: Content-addressed build cache with atomic writes.
// Usage: packBundle owns the cache directory and retains only its current payloads.
// Related: devtools/cmd/uiassets/bundle.go.
package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func compressedAsset(source []byte, hash [sha256.Size]byte, path string) ([]byte, error) {
	cached, err := os.ReadFile(path)
	if err == nil {
		reader, openErr := gzip.NewReader(bytes.NewReader(cached))
		if openErr == nil {
			digest := sha256.New()
			size, readErr := io.Copy(digest, io.LimitReader(reader, int64(len(source))+1))
			closeErr := reader.Close()
			if readErr == nil && closeErr == nil && size == int64(len(source)) && bytes.Equal(digest.Sum(nil), hash[:]) {
				return cached, nil
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	var payload bytes.Buffer
	writer, err := gzip.NewWriterLevel(&payload, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	_, writeErr := writer.Write(source)
	if err := errors.Join(writeErr, writer.Close()); err != nil {
		return nil, err
	}
	if err := atomicWrite(path, payload.Bytes()); err != nil {
		return nil, err
	}
	return payload.Bytes(), nil
}

func atomicWrite(path string, body []byte) error {
	previous, err := os.ReadFile(path)
	if err == nil && bytes.Equal(previous, body) {
		return nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".meca-ui-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, writeErr := file.Write(body)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func pruneCache(directory string, keep map[string][]byte) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".gz") {
			continue
		}
		hash, err := hex.DecodeString(strings.TrimSuffix(name, ".gz"))
		if err != nil || len(hash) != sha256.Size {
			continue
		}
		if _, exists := keep[name]; !exists {
			if err := os.Remove(filepath.Join(directory, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	return nil
}
