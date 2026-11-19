// Purpose: Verify deduplication invalidation, empty observations and browser admission release.
// Pattern: Focused memory ownership and observation contract regressions.
// Usage: go test -race ./internal/tools/browser.
// Related: internal/tools/browser/results.go, internal/tools/browser/admission.go.
package browser

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/keonho-kim/meca/internal/processmemory"
	browserruntime "github.com/keonho-kim/meca/internal/tools/browser/runtime"
)

func TestUnchangedObservationSkipsBodyProjection(t *testing.T) {
	calls, version := 0, 0
	s := New("", "", func(text string) string {
		if text == "body" {
			calls++
		}
		return text
	}, func() int { return version }, nil)

	s.work = t.TempDir()
	raw := json.RawMessage(`{"text":"body"}`)
	for range 4 {
		if _, err := s.store(raw, 0); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatal("unchanged body projected repeatedly", calls)
	}
	version++
	if _, err := s.store(raw, 0); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatal("mask version did not invalidate projection", calls)
	}
	empty, err := s.store(json.RawMessage(`{"text":""}`), 0)
	if err != nil || empty.Text != "" || empty.TotalBytes != 0 {
		t.Fatal("empty text replaced with JSON", empty, err)
	}
	if _, err := s.store(json.RawMessage(`{"text":42}`), 0); err == nil {
		t.Fatal("malformed text accepted")
	}
}

func TestBrowserAdmissionReleasesFailedStart(t *testing.T) {
	slots, err := browserCapacity()
	if err != nil {
		t.Fatal(err)
	}
	s := New("", "", func(text string) string { return text }, nil, nil)
	s.prepare = func(context.Context, string) (browserruntime.Paths, error) {
		return browserruntime.Paths{}, errors.New("fixture failure")
	}
	for range cap(slots) + 1 {
		if err := s.start(t.Context()); err == nil {
			t.Fatal("expected failed start")
		}
	}
	if len(slots) != 0 {
		t.Fatal("failed start retained capacity")
	}
	var releases []func()
	defer func() {
		for _, release := range releases {
			release()
		}
	}()
	for range cap(slots) {
		release, err := acquireBrowser()
		if err != nil {
			t.Fatal(err)
		}
		releases = append(releases, release)
	}
	if _, err := acquireBrowser(); err == nil {
		t.Fatal("external browser admission unbounded")
	}
	releases[0]()
	releases[0]()
	release, err := acquireBrowser()
	if err != nil {
		t.Fatal("capacity not reusable", err)
	}
	release()
}

func TestImageReservationSurvivesDeliveryUntilClose(t *testing.T) {
	lease := processmemory.NewPayloadLease()
	if err := lease.Resize(1024); err != nil {
		t.Fatal(err)
	}
	page := Page{Image: make([]byte, 1024), memory: lease}
	defer page.Close()
	blocker := processmemory.NewPayloadLease()
	defer blocker.Close()
	if err := blocker.Resize(int(processmemory.PayloadBytes)); err == nil {
		t.Fatal("image handoff was unaccounted")
	}
	page.Close()
	page.Close()
	if len(page.Image) != 0 {
		t.Fatal("closed image retained bytes")
	}
	if err := blocker.Resize(int(processmemory.PayloadBytes)); err != nil {
		t.Fatal(err)
	}
}
