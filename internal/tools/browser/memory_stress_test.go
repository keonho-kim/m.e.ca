// Purpose: Probe live heap after repeated and concurrent large browser observations and bounded artifact eviction.
// Pattern: Deterministic retention contracts plus opt-in diagnostic heap sampling.
// Usage: go test ./internal/tools/browser -run TestBrowserMemory -v; set MECA_BROWSER_MEMORY_PROBE=1 for heap reports.
// Related: internal/tools/browser/results.go, internal/processmemory/payload.go.
package browser

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"runtime/metrics"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/processmemory"
)

func TestBrowserMemoryRetention(t *testing.T) {
	s := New("", "", func(text string) string { return text }, nil, nil)
	s.work = t.TempDir()
	raw, _ := json.Marshal(map[string]string{"text": strings.Repeat("x", 3<<20)})
	first, err := s.store(raw, 0)
	if err != nil {
		t.Fatal(err)
	}
	for range 32 {
		page, err := s.store(raw, 0)
		if err != nil || !page.Unchanged || page.ResultID != first.ResultID || page.Text != "" {
			t.Fatal(page, err)
		}
	}
	for i := range 32 {
		s.setAddress(fmt.Sprintf("http://fixture/%d", i))
		if _, err := s.store(raw, 0); err != nil {
			t.Fatal(err)
		}
	}
	files, err := os.ReadDir(s.work)
	if err != nil {
		t.Fatal(err)
	}
	var bytes int64
	err = filepath.WalkDir(s.work, func(_ string, file fs.DirEntry, err error) error {
		if err != nil || file.IsDir() {
			return err
		}
		info, err := file.Info()
		if err != nil {
			return err
		}
		bytes += info.Size()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 10 || len(s.results) != len(files) || bytes != 10*((3<<20)-(2<<10)) {
		t.Fatal("retained artifact budget exceeded", len(files), bytes)
	}
}

func TestBrowserMemoryAdmission(t *testing.T) {
	blocker := processmemory.NewPayloadLease()
	defer blocker.Close()
	if err := blocker.Resize(int(processmemory.PayloadBytes)); err != nil {
		t.Fatal(err)
	}
	s := New("", "", func(text string) string { return text }, nil, nil)
	s.work = t.TempDir()
	raw := json.RawMessage(`{"text":"bounded"}`)
	if _, err := s.store(raw, 0); err == nil {
		t.Fatal("observation ignored exhausted memory admission")
	}
	if _, err := s.invoke(t.Context(), "snapshot"); err == nil {
		t.Fatal("capture ignored exhausted memory admission")
	}
	if _, err := s.screenshot(t.Context()); err == nil {
		t.Fatal("screenshot ignored exhausted memory admission")
	}
	files, err := os.ReadDir(s.work)
	if err != nil || len(files) != 0 || len(s.results) != 0 {
		t.Fatal("refused observation retained artifacts", len(files), err)
	}
	blocker.Close()
	if _, err := s.store(raw, 0); err != nil {
		t.Fatal("observation failed after memory release", err)
	}
	probe := processmemory.NewPayloadLease()
	defer probe.Close()
	if err := probe.Resize(int(processmemory.PayloadBytes)); err != nil {
		t.Fatal("completed observation retained a reservation", err)
	}
}

func TestBrowserMemoryProbe(t *testing.T) {
	if os.Getenv("MECA_BROWSER_MEMORY_PROBE") != "1" {
		t.Skip("opt-in heap observation")
	}
	for _, workers := range []int{1, 8} {
		t.Run(fmt.Sprintf("workers-%d", workers), func(t *testing.T) {
			raw, _ := json.Marshal(map[string]string{"text": strings.Repeat("x", 3<<20)})
			sessions := make([]*Session, workers)
			for i := range sessions {
				sessions[i] = New("", "", func(text string) string { return text }, nil, nil)
				sessions[i].work = t.TempDir()
			}
			// Drain encoding/json's setup pools before taking the baseline.
			runtime.GC()
			runtime.GC()
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			stop, sampled := make(chan struct{}), make(chan uint64, 1)
			go func() {
				ticker := time.NewTicker(2 * time.Millisecond)
				defer ticker.Stop()
				peak := before.HeapAlloc
				sample := []metrics.Sample{{Name: "/memory/classes/heap/objects:bytes"}}
				for {
					select {
					case <-ticker.C:
						metrics.Read(sample)
						peak = max(peak, sample[0].Value.Uint64())
					case <-stop:
						sampled <- peak
						return
					}
				}
			}()
			var group sync.WaitGroup
			for _, s := range sessions {
				group.Go(func() {
					for range 128 {
						if _, err := s.store(raw, 0); err != nil {
							t.Error(err)
							return
						}
					}
				})
			}
			group.Wait()
			close(stop)
			peak := <-sampled
			runtime.GC()
			runtime.ReadMemStats(&after)
			runtime.KeepAlive(sessions)
			runtime.KeepAlive(raw)
			t.Logf("total_alloc=%d bytes; sampled_heap_peak_delta=%d bytes; retained_heap_delta=%d bytes; mallocs=%d; GC_cycles=%d", after.TotalAlloc-before.TotalAlloc, peak-before.HeapAlloc, int64(after.HeapAlloc)-int64(before.HeapAlloc), after.Mallocs-before.Mallocs, after.NumGC-before.NumGC)
		})
	}
}
