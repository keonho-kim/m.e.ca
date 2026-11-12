// Purpose: Verify shared directory observations retain independent bounded subscriptions and join shutdown.
// Pattern: Resource-lifecycle regression tests.
// Usage: go test -race ./internal/fileview -run TestDirectoryWatch.
// Related: internal/fileview/watch_pool.go.
package fileview

import (
	"slices"
	"testing"
)

func TestDirectoryWatchSharedLifetime(t *testing.T) {
	var pool directoryWatches
	path := t.TempDir()
	first, _, releaseFirst, err := pool.borrow(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(releaseFirst)
	second, updates, releaseSecond, err := pool.borrow(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(releaseSecond)
	if first != second || len(pool.entries) != 1 {
		t.Fatal("duplicate directory watcher")
	}
	releaseFirst()
	select {
	case <-second.done:
		t.Fatal("remaining subscriber lost its watcher")
	default:
	}
	pool.publish(second, FileChange{Refresh: true, Names: []string{"remaining.txt"}})
	if change := <-updates; !slices.Contains(change.Names, "remaining.txt") {
		t.Fatal(change)
	}
	releaseSecond()
	select {
	case <-second.done:
	default:
		t.Fatal("shutdown did not join watcher")
	}
	if len(pool.entries) != 0 {
		t.Fatal("released watcher retained")
	}
}

func TestDirectoryWatchCoalescesEachSubscriber(t *testing.T) {
	var pool directoryWatches
	entry := &directoryWatch{subscribers: make(map[chan FileChange]struct{})}
	one, two := make(chan FileChange, 1), make(chan FileChange, 1)
	entry.subscribers[one], entry.subscribers[two] = struct{}{}, struct{}{}
	one <- FileChange{Refresh: true, Names: []string{"first.txt"}}
	two <- FileChange{Refresh: true, Names: []string{"second.txt"}}
	pool.publish(entry, FileChange{Refresh: true, Names: []string{"new.txt"}})
	for channel, own := range map[chan FileChange]string{one: "first.txt", two: "second.txt"} {
		change := <-channel
		if !slices.Equal(change.Names, []string{own, "new.txt"}) && !slices.Equal(change.Names, []string{"new.txt", own}) {
			t.Fatal(change)
		}
	}
}
