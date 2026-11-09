// Purpose: Own diagnostic output failure messages and bounded queue policy.
// Pattern: Declarative diagnostic policy.
// Usage: Sink and private file adapters share these fixed messages and limits.
// Related: internal/diagnostics/logger.go, internal/diagnostics/logfile/file.go.
package catalog

import "time"

const (
	DiagnosticWriteFailed          Message = "diagnostic log write failed; execution records remain authoritative"
	DiagnosticDependenciesRequired Message = "diagnostic writer and redactor are required"
	DiagnosticRecordLimit          Message = "diagnostic record exceeds file limit"
	DiagnosticDirectoryLimit       Message = "diagnostic directory entry limit exceeded"
	DiagnosticRecordLimitCode              = "diagnostic_record_limit"
	LogWriteFailed                         = "log_write_failed"
	RotationDeferredCode                   = "rotation_deferred"
	LogDrainTimeout                        = 2 * time.Second
	LogSummaryInterval                     = time.Minute
)
