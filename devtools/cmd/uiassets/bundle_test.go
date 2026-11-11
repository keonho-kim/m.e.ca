// Purpose: Verify reproducible, cache-aware browser bundles survive concurrent source reads.
// Pattern: Build-artifact and failure-boundary integration tests.
// Usage: go -C devtools test -race ./cmd/uiassets.
// Related: devtools/cmd/uiassets/bundle.go, devtools/cmd/uiassets/compression.go.
package main

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func readPublished(source []byte) (string, error) {
	tree, err := parser.ParseFile(token.NewFileSet(), "bundle_generated.go", source, 0)
	if err != nil {
		return "", err
	}
	var encoded string
	ast.Inspect(tree, func(node ast.Node) bool {
		if literal, ok := node.(*ast.BasicLit); ok && literal.Kind == token.STRING {
			encoded, err = strconv.Unquote(literal.Value)
		}
		return true
	})
	if err != nil {
		return "", err
	}
	archive, err := zip.NewReader(strings.NewReader(encoded), int64(len(encoded)))
	if err != nil {
		return "", err
	}
	metadata, err := fs.ReadFile(archive, "manifest.json")
	if err != nil {
		return "", err
	}
	var manifest map[string]asset
	if err := json.Unmarshal(metadata, &manifest); err != nil {
		return "", err
	}
	for _, entry := range archive.File {
		if entry.Method != zip.Store {
			return "", fmt.Errorf("gzip payload was compressed twice")
		}
	}
	payload, err := fs.ReadFile(archive, manifest["index.html"].File)
	if err != nil {
		return "", err
	}
	reader, err := gzip.NewReader(bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	body, err := io.ReadAll(reader)
	closeErr := reader.Close()
	if err == nil {
		err = closeErr
	}
	if len(body) != manifest["index.html"].Bytes {
		return "", fmt.Errorf("manifest and payload disagree")
	}
	return string(body), err
}

func TestBundleCacheAndPublication(t *testing.T) {
	base := t.TempDir()
	input, output, cache := filepath.Join(base, "dist"), filepath.Join(base, "bundle_generated.go"), filepath.Join(base, "cache")
	if err := os.Mkdir(input, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(input, "index.html"), []byte("<html>first</html>"), 0600); err != nil {
		t.Fatal(err)
	}
	if count, err := packBundle(input, output, cache); err != nil || count != 1 {
		t.Fatal(count, err)
	}
	first, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if text, err := readPublished(first); err != nil || text != "<html>first</html>" {
		t.Fatal(text, err)
	}
	entries, err := os.ReadDir(cache)
	if err != nil || len(entries) != 1 {
		t.Fatal(entries, err)
	}
	cached := filepath.Join(cache, entries[0].Name())
	old := time.Unix(1000, 0)
	for _, path := range []string{output, cached} {
		if err := os.Chtimes(path, old, old); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := packBundle(input, output, cache); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{output, cached} {
		info, err := os.Stat(path)
		if err != nil || !info.ModTime().Equal(old) {
			t.Fatalf("unchanged build rewrote %s: %v", path, err)
		}
	}
	if err := os.WriteFile(cached, []byte("corrupt cache"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := packBundle(input, output, cache); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(input, "index.html"), []byte("<html>second</html>"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := packBundle(input, output, cache); err != nil {
		t.Fatal(err)
	}
	if entries, err := os.ReadDir(cache); err != nil || len(entries) != 1 {
		t.Fatal("obsolete compression cache retained", entries, err)
	}
	second, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if text, err := readPublished(second); err != nil || text != "<html>second</html>" {
		t.Fatal(text, err)
	}
	if err := os.Remove(filepath.Join(input, "index.html")); err != nil {
		t.Fatal(err)
	}
	if _, err := packBundle(input, output, cache); err == nil {
		t.Fatal("incomplete browser build published")
	}
	current, err := os.ReadFile(output)
	if err != nil || !bytes.Equal(second, current) {
		t.Fatal("failed build changed the published bundle", err)
	}
}

func TestBundleReadersObserveCompleteGenerations(t *testing.T) {
	base := t.TempDir()
	input, output, cache := filepath.Join(base, "dist"), filepath.Join(base, "bundle_generated.go"), filepath.Join(base, "cache")
	if err := os.Mkdir(input, 0700); err != nil {
		t.Fatal(err)
	}
	write := func(value string) error {
		if err := os.WriteFile(filepath.Join(input, "index.html"), []byte(value), 0600); err != nil {
			return err
		}
		_, err := packBundle(input, output, cache)
		return err
	}
	if err := write("first"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	errors := make(chan error, 1)
	var readers sync.WaitGroup
	readers.Go(func() {
		for ctx.Err() == nil {
			source, err := os.ReadFile(output)
			if err == nil {
				_, err = readPublished(source)
			}
			if err != nil {
				errors <- err
				return
			}
		}
	})
	for index := range 12 {
		if err := write(fmt.Sprintf("generation %d", index)); err != nil {
			cancel()
			readers.Wait()
			t.Fatal(err)
		}
	}
	cancel()
	readers.Wait()
	select {
	case err := <-errors:
		t.Fatal("reader observed a partial build", err)
	default:
	}
}
