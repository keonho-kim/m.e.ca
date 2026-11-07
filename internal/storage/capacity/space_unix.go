//go:build linux || darwin

// Purpose: Keep emergency filesystem space before admitting new persistent allocations.
// Pattern: Native filesystem capacity boundary.
// Usage: Allocation reservations check the owning filesystem without deleting user state.
// Related: internal/storage/capacity/reserve.go.
package capacity

import "syscall"

func CheckFree(path string, minimum uint64) error {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return err
	}
	if stat.Bsize <= 0 {
		return ErrLimit
	}
	block := uint64(stat.Bsize)
	needed := minimum / block
	if minimum%block != 0 {
		needed++
	}
	if uint64(stat.Bavail) < needed {
		return ErrLimit
	}
	return nil
}
