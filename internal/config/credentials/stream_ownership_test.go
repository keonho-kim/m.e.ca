// Purpose: Verify lazy sanitized copies preserve credential boundaries, invalid UTF-8 and old snapshots.
// Pattern: Stream ownership and sanitization regression tests.
// Usage: go test ./internal/config/credentials -run TestStreamMask -bench BenchmarkCleanStreamMask.
// Related: internal/config/credentials/stream.go, internal/config/credentials/redaction.go.
package credentials

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestStreamMaskLazyCopyPreservesOldSnapshots(t *testing.T) {
	v := &Vault{}
	v.Remember("provider-secret")
	mask := v.NewStreamMask()
	var raw strings.Builder
	raw.WriteString("한글 😀 prefix ")
	old := mask(raw.String(), false)
	raw.WriteString("provider-sec")
	if got := mask(raw.String(), false); got != old {
		t.Fatal("partial credential leaked")
	}
	raw.WriteString("ret suffix\x1b")
	if got := mask(raw.String(), true); got != "한글 😀 prefix [redacted] suffix" {
		t.Fatal(got)
	}
	if old != "한글 😀 prefix " {
		t.Fatal("later masking changed an earlier snapshot")
	}
}

func TestStreamMaskLazyCopyHandlesInvalidRunesAndShortening(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"clean 한글 👨‍👩‍👧‍👦\t\n", "clean 한글 👨‍👩‍👧‍👦\t\n"},
		{"prefix\x00body\x7f", "prefixbody"},
		{"\xffbody\xe2\x82", "�body��"},
	} {
		mask := (&Vault{}).NewStreamMask()
		for at := 1; at <= len(tc.input); at++ {
			if got := mask(tc.input[:at], false); !utf8.ValidString(got) {
				t.Fatal("partial UTF-8 escaped")
			}
		}
		if got := mask(tc.input, true); got != tc.want {
			t.Fatalf("got %q want %q", got, tc.want)
		}
		if got := mask("new", true); got != "new" {
			t.Fatal("shortened response retained prior output")
		}
	}
}

func TestStreamMaskLazyCopyKeepsLongestCredentialAndUpdates(t *testing.T) {
	v := &Vault{}
	v.Remember("key-short")
	v.Remember("key-short-long")
	mask := v.NewStreamMask()
	if got := mask("safe key-short", false); got != "safe " {
		t.Fatal("prefix of longer credential exposed", got)
	}
	if got := mask("safe key-short-long!", true); got != "safe [redacted]!" {
		t.Fatal(got)
	}
	v.Remember("safe")
	if got := mask("safe key-short-long!!", true); got != "[redacted] [redacted]!!" {
		t.Fatal("new credential did not invalidate clean prefix", got)
	}
}

func BenchmarkCleanStreamMask(b *testing.B) {
	v := &Vault{}
	v.Remember("credential-0-" + strings.Repeat("s", 32))
	text := strings.Repeat("token fragment. ", (1<<20)/16)
	newMask := v.NewStreamMask
	b.ReportAllocs()
	for b.Loop() {
		mask := newMask()
		for at := 64; at <= len(text); at += 64 {
			mask(text[:at], false)
		}
		if mask(text, true) != text {
			b.Fatal("clean text changed")
		}
	}
}
