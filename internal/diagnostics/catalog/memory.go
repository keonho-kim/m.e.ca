// Purpose: Own transient process-memory admission diagnostics.
// Pattern: Declarative diagnostic catalog.
// Usage: Reply memory admission reports one stable resource failure.
// Related: internal/processmemory/transient.go.
package catalog

const ReplyMemoryLimit Message = "aggregate in-flight model replies exceed their process byte budget"

const PayloadMemoryLimit Message = "selected originals exceed the available process payload budget"
