// Purpose: Own file-mutation admission, receipt and storage diagnostics.
// Pattern: Central diagnostic catalog.
// Usage: Native file mutations and their private receipt ledger reference these messages.
// Related: internal/commands/server/file_mutation_receipt.go, internal/storage/filemutations.
package catalog

const (
	FileMutationKey      Message = "provide an 8 to 128 byte mutation key for this operation"
	FileMutationConflict Message = "the mutation key belongs to a different file operation"
	FileMutationStorage  Message = "unsupported file mutation receipt storage; original records were preserved"
	FileMutationCapacity Message = "file mutation receipt capacity is exhausted"
)
