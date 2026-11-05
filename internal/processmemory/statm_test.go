// Purpose: Verify resident-page parsing handles OS page sizes and rejects unusable snapshots.
// Pattern: Pure input-boundary tests.
// Usage: go test ./internal/processmemory.
// Related: internal/processmemory/statm.go, internal/processmemory/resident_linux.go.
package processmemory

import "testing"

func TestResidentPages(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		pageSize   uint64
		want       uint64
		invalid    bool
	}{
		{"4KiB pages", "1000 42 7 2 0 3 0\n", 4096, 42 * 4096, false},
		{"16KiB pages", "1000\t42 7 2 0 3 0\n", 16384, 42 * 16384, false},
		{"missing", "1000", 4096, 0, true},
		{"not a count", "1000 invalid", 4096, 0, true},
		{"negative", "1000 -1", 4096, 0, true},
		{"overflow", "1000 18446744073709551615", 4096, 0, true},
		{"invalid page size", "1000 42", 0, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := residentPages([]byte(tc.text), tc.pageSize)
			if (err != nil) != tc.invalid || got != tc.want {
				t.Fatalf("bytes=%d error=%v; want bytes=%d invalid=%t", got, err, tc.want, tc.invalid)
			}
		})
	}
}
