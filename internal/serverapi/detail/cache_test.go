// Purpose: Verify one-load paging, masking revisions and private spool cleanup in the API detail adapter.
// Pattern: Deterministic cache and resource lifecycle tests.
// Usage: go test -race ./internal/serverapi/detail -run TestCache.
// Related: internal/serverapi/detail/cache.go, internal/serverapi/detail/document.go.
package detail

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/config/credentials"
)

func cacheFixture(t testing.TB, value *record.Event) (*Cache, Request, *atomic.Int64, *credentials.Vault) {
	t.Helper()
	c := NewCache()
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	})
	vault := &credentials.Vault{}
	loads := &atomic.Int64{}
	r := Request{Session: "selected-session", Source: server.DetailSource{Tool: record.EventReference{Sequence: 1}}, Version: vault.RedactionVersion, NewMask: vault.NewChunkMask, Load: func(func(int) error) (server.DetailOriginal, error) {
		loads.Add(1)
		return server.DetailOriginal{Tool: value}, nil
	}}
	return c, r, loads, vault
}

func TestCachePagesLoadOnceAndRefreshMask(t *testing.T) {
	value := &record.Event{Kind: "tool_succeeded", Text: strings.Repeat("x", 128<<10) + "future-key"}
	c, request, loads, vault := cacheFixture(t, value)
	var joined []byte
	for {
		page, total, err := c.Read(t.Context(), request, int64(len(joined)), 64<<10)
		if err != nil {
			t.Fatal(err)
		}
		joined = append(joined, page...)
		if int64(len(joined)) == total {
			break
		}
	}
	if !json.Valid(joined) || !bytes.Contains(joined, []byte("future-key")) || loads.Load() != 1 {
		t.Fatal("original was not paged once")
	}
	vault.Remember("future-key")
	page, _, err := c.Read(t.Context(), request, int64(len(joined)-128), 1024)
	if err != nil || bytes.Contains(page, []byte("future-key")) || !bytes.Contains(page, []byte("[redacted]")) || loads.Load() != 2 {
		t.Fatal(string(page), loads.Load(), err)
	}
}

func TestCacheConcurrentPagesLoadOnce(t *testing.T) {
	c, request, loads, _ := cacheFixture(t, &record.Event{Text: strings.Repeat("x", 128<<10)})
	var workers sync.WaitGroup
	for i := range 4 {
		workers.Go(func() {
			if _, _, err := c.Read(t.Context(), request, int64(i*1024), 1024); err != nil {
				t.Error(err)
			}
		})
	}
	workers.Wait()
	if loads.Load() != 1 {
		t.Fatal("concurrent paging reloaded the original", loads.Load())
	}
}

func TestCacheSpoolFollowsSelectedSession(t *testing.T) {
	value := &record.Event{Text: strings.Repeat("x", 128<<10)}
	c, request, _, _ := cacheFixture(t, value)
	if _, _, err := c.Read(t.Context(), request, 0, 1024); err != nil {
		t.Fatal(err)
	}
	first := c.file
	if first == nil {
		t.Fatal("large detail retained in memory")
	}
	if _, err := os.Stat(first.Name()); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("spool retained a pathname", err)
	}
	if info, err := first.Stat(); err != nil || info.Mode().Perm() != 0600 {
		t.Fatal(info, err)
	}
	value.Text, request.Source.Tool.Sequence = "small", 2
	if _, _, err := c.Read(t.Context(), request, 0, 1024); err != nil {
		t.Fatal(err)
	}
	if _, err := first.Stat(); !errors.Is(err, os.ErrClosed) {
		t.Fatal("replaced spool stayed open", err)
	}
	value.Text, request.Source.Tool.Sequence = strings.Repeat("y", 128<<10), 3
	if _, _, err := c.Read(t.Context(), request, 0, 1024); err != nil {
		t.Fatal(err)
	}
	last := c.file
	if err := c.CloseSession("another-session"); err != nil {
		t.Fatal(err)
	}
	if _, err := last.Stat(); err != nil {
		t.Fatal("unrelated session closed this spool", err)
	}
	if err := c.CloseSession(request.Session); err != nil {
		t.Fatal(err)
	}
	if _, err := last.Stat(); !errors.Is(err, os.ErrClosed) {
		t.Fatal("session close retained its spool", err)
	}
}

func TestCacheWorksWithoutTemporaryStorage(t *testing.T) {
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "unavailable"))
	c, request, loads, _ := cacheFixture(t, &record.Event{Text: strings.Repeat("x", 128<<10)})
	for _, offset := range []int64{0, 1024} {
		page, total, err := c.Read(t.Context(), request, offset, 1024)
		if err != nil || len(page) != 1024 || total <= 128<<10 {
			t.Fatal(len(page), total, err)
		}
	}
	if loads.Load() != 2 {
		t.Fatal("uncached original did not reload")
	}
}

func TestCacheWaitingReaderCanCancel(t *testing.T) {
	c, request, _, _ := cacheFixture(t, &record.Event{Text: "visible"})
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	load := request.Load
	request.Load = func(reserve func(int) error) (server.DetailOriginal, error) {
		close(entered)
		<-release
		return load(reserve)
	}
	go func() { _, _, err := c.Read(t.Context(), request, 0, 1024); done <- err }()
	<-entered
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, _, err := c.Read(ctx, request, 0, 1024)
	close(release)
	if cause := <-done; cause != nil {
		t.Fatal(cause)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func BenchmarkDetailPages(b *testing.B) {
	c, request, loads, _ := cacheFixture(b, &record.Event{Text: strings.Repeat("x", 1<<20)})
	b.ReportAllocs()
	b.SetBytes(1 << 20)
	for b.Loop() {
		request.Source.Tool.Sequence++
		var offset int64
		for {
			page, total, err := c.Read(b.Context(), request, offset, 64<<10)
			if err != nil {
				b.Fatal(err)
			}
			offset += int64(len(page))
			if offset == total {
				break
			}
		}
	}
	b.ReportMetric(float64(loads.Load())/float64(b.N), "loads/op")
}
