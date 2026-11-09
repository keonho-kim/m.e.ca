// Purpose: Normalize SQLite primary failure codes into stable product diagnostics.
// Pattern: Declarative persistence failure catalog.
// Usage: The central classifier consumes driver Code() values without importing storage adapters.
// Related: internal/diagnostics/fault/classify.go, internal/diagnostics/catalog/semantic.go.
package catalog

const (
	StorageBusy                    = "storage_busy"
	StorageReadOnly                = "storage_read_only"
	StorageFull                    = "storage_full"
	StorageCorrupt                 = "storage_corrupt"
	StorageIO                      = "storage_io_failed"
	StorageBusyMessage     Message = "storage is busy; inspect recorded task state before retrying"
	StorageReadOnlyMessage Message = "storage is read-only; restore write access before starting more work"
	StorageFullMessage     Message = "storage is full; preserve existing records and free space before retrying"
	StorageCorruptMessage  Message = "storage integrity could not be verified; preserve the database and its backups"
)

func SQLiteCode(code int) string {
	switch code & 255 {
	case 5, 6:
		return StorageBusy
	case 8:
		return StorageReadOnly
	case 13:
		return StorageFull
	case 11, 26:
		return StorageCorrupt
	case 10:
		return StorageIO
	case 14:
		return Unavailable
	default:
		return ""
	}
}
