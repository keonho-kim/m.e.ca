// Purpose: Preserve complete integrity, cancellation and JSON semantics during bounded preview decoding.
// Pattern: Codec differential tests and fragmented-input fuzzing.
// Usage: go test ./internal/storage/content -run TestPrefix -fuzz FuzzPrefixString.
// Related: internal/storage/content/prefix_reader.go, internal/storage/content/prefix_string.go.
package content

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestPrefixCodecsVerifyTheEntireOriginal(t *testing.T) {
	var codec Codec
	defer codec.Close()
	for _, raw := range [][]byte{[]byte(`""`), []byte(` "a\uD83D\uDE00\n한글" `), append([]byte(`"`), append(bytes.Repeat([]byte("x"), 128<<10), '"')...)} {
		compressed, err := codec.Encode(raw)
		if err != nil {
			t.Fatal(err)
		}
		var zipped bytes.Buffer
		writer := gzip.NewWriter(&zipped)
		if _, err := writer.Write(raw); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		for kind, body := range map[int][]byte{0: raw, 1: zipped.Bytes(), 2: compressed} {
			if kind == 2 && len(raw) <= Threshold {
				continue
			} // Only large values enter the stored Zstandard format.
			for _, limit := range []int{1, 4, 16, 16384} {
				want, truncated, err := StringPrefix(raw, limit)
				if err != nil {
					t.Fatal(err)
				}
				got, clipped, err := codec.TextPrefix(t.Context(), kind, len(raw), ID(raw), body, limit)
				if err != nil || got != want || clipped != truncated {
					t.Fatalf("codec=%d limit=%d: %q %v %v; want %q %v", kind, limit, got, clipped, err, want, truncated)
				}
				if _, _, err := codec.TextPrefix(t.Context(), kind, len(raw), strings.Repeat("0", 64), body, limit); err == nil {
					t.Fatal("incorrect full digest accepted")
				}
			}
		}
		if len(raw) > Threshold {
			if _, err := codec.Restore(2, len(raw), ID(raw), compressed); err != nil {
				t.Fatal("stream reset broke full restoration", err)
			}
		}
	}
}

func TestPrefixRejectsInvalidTailAndCancellation(t *testing.T) {
	for _, raw := range []string{`"visible" garbage`, `"visible\q"`, `"visible\u00x0"`, `"unfinished`, "\"raw\ncontrol\""} {
		if _, _, err := readProjection(t.Context(), strings.NewReader(raw), len(raw), ID([]byte(raw)), 1, false); err == nil {
			t.Fatal("invalid tail accepted", raw)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	raw := []byte(`"` + strings.Repeat("x", 128<<10) + `"`)
	reader := cancelPrefixReader{Reader: bytes.NewReader(raw), cancel: cancel}
	if text, _, err := readProjection(ctx, reader, len(raw), ID(raw), 16, false); !errors.Is(err, context.Canceled) || text != "" {
		t.Fatal("partial text escaped cancellation", text, err)
	}
	if _, _, err := readProjection(t.Context(), bytes.NewReader(raw), len(raw)-1, ID(raw), 16, false); err == nil {
		t.Fatal("oversized original accepted")
	}
}

type cancelPrefixReader struct {
	io.Reader
	cancel context.CancelFunc
}

func (r cancelPrefixReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.cancel()
	return n, err
}

func FuzzPrefixString(f *testing.F) {
	for _, raw := range []string{`""`, `"hello"`, `"\u0000\u0000\uD83D\uDE00한글"`, ` "abc\\\"def" `, `"abc\q"`, "\"\xff\x80\"", `"` + strings.Repeat(`\u0000`, 100) + `"`} {
		f.Add([]byte(raw), uint8(1))
	}
	f.Fuzz(func(t *testing.T, raw []byte, bound uint8) {
		if len(raw) > 128<<10 {
			t.Skip()
		}
		limit := int(bound) + 1
		want, clipped, wantErr := StringPrefix(raw, limit)
		if !json.Valid(raw) {
			wantErr = errors.New("invalid complete JSON string")
		}
		for _, chunk := range []int{1, 7, 32768} {
			parser := newPrefixString(len(raw), limit)
			var err error
			for offset := 0; offset < len(raw) && err == nil; offset += chunk {
				err = parser.write(raw[offset:min(offset+chunk, len(raw))])
			}
			var got string
			var truncated bool
			if err == nil {
				got, truncated, err = parser.finish(limit)
			}
			if (err != nil) != (wantErr != nil) || err == nil && (got != want || truncated != clipped) {
				t.Fatalf("chunk=%d raw=%q got=%q,%v,%v want=%q,%v,%v", chunk, raw, got, truncated, err, want, clipped, wantErr)
			}
		}
	})
}
