// Purpose: Compose mode-appropriate bounded diagnostics without mixing logs into model output.
// Pattern: Process composition root.
// Usage: Normal startup joins Sink before releasing its optional private TUI file.
// Related: cmd/meca/main.go, internal/diagnostics/level.go, internal/diagnostics/logfile/file.go.
package main

import (
	"context"
	"errors"

	"io"
	"os"
	"path/filepath"

	"github.com/keonho-kim/meca/internal/commands/startup"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/diagnostics"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/diagnostics/logfile"
)

func loggingContext(ctx context.Context, state string, mode startup.Mode) (context.Context, func() error, error) {
	level, err := diagnostics.ParseLevel(os.Getenv("MECA_LOG_LEVEL"))
	if err != nil {
		return nil, nil, err
	}
	output, err := diagnostics.OpenStderr()
	if err != nil {
		return nil, nil, err
	}
	var writer io.Writer = output
	interrupt := output.Close
	var file *logfile.File
	if !mode.Print && !mode.Server && !mode.Export {
		var err error
		file, err = logfile.OpenFile(ctx, filepath.Join(state, "logs"))
		if err != nil {
			return nil, nil, errors.Join(faults.Errorf(diagcatalog.MecaOpenPrivateDiagnosticLog, err), output.Close())
		}
		writer = file
		interrupt = file.Interrupt
		_ = output.Close()
	}
	vault := &credentials.Vault{State: state}
	sink, err := diagnostics.New(diagnostics.Options{Writer: writer, Level: level, Redact: vault.Redact, Interrupt: interrupt})
	if err != nil {
		if file != nil {
			err = errors.Join(err, file.Close())
		} else {
			err = errors.Join(err, output.Close())
		}
		return nil, nil, err
	}
	return diagnostics.WithSink(ctx, sink), func() error {
		err := sink.Close()
		if file == nil {
			err = errors.Join(err, output.Close())
		}
		if file != nil {
			err = errors.Join(err, file.Close())
		}
		return err
	}, nil
}
