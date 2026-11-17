// Purpose: Measure download allocations as runtime payload size grows without retaining the source in memory.
// Pattern: Local streaming HTTP benchmark with on-disk digest verification.
// Usage: go test ./internal/tools/browser/runtime -run '^$' -bench BenchmarkRuntimeDownloadMemory -benchmem.
// Related: internal/tools/browser/runtime/download.go.
package runtime

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func BenchmarkRuntimeDownloadMemory(b *testing.B) {
	for _, size := range []int{1 << 20, 64 << 20} {
		b.Run(fmt.Sprintf("%dMiB", size>>20), func(b *testing.B) {
			chunk := bytes.Repeat([]byte("x"), 32<<10)
			digest := sha256.New()
			for offset := 0; offset < size; offset += len(chunk) {
				digest.Write(chunk[:min(len(chunk), size-offset)])
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", strconv.Itoa(size))
				for offset := 0; offset < size; offset += len(chunk) {
					if _, err := w.Write(chunk[:min(len(chunk), size-offset)]); err != nil {
						return
					}
				}
			}))
			defer server.Close()
			value := asset{url: server.URL, digest: hex.EncodeToString(digest.Sum(nil))}
			path := filepath.Join(b.TempDir(), "download")
			b.ReportAllocs()
			b.SetBytes(int64(size))
			b.ResetTimer()
			for b.Loop() {
				if err := download(b.Context(), server.Client(), value, path); err != nil {
					b.Fatal(err)
				}
				if err := os.Remove(path); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
