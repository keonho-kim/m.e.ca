// Purpose: Extract one regular MECA executable without publishing archive-controlled paths or links.
// Pattern: Filesystem input boundary.
// Usage: fetch invokes extract only after verifying the complete archive checksum.
// Related: internal/commands/upgrade/release.go, internal/commands/upgrade/upgrade.go.
package upgrade

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
)

func extract(directory *os.Root) (err error) {
	archive, err := directory.Open("release.tar.gz")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, archive.Close()) }()
	compressed, err := gzip.NewReader(archive)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, compressed.Close()) }()
	expanded := &io.LimitedReader{R: compressed, N: 2*maxArchiveBytes + 1}
	reader := tar.NewReader(expanded)
	found := false
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if header.Name != "meca" {
			continue // License files and other entries are never written to disk.
		}
		if found || header.Typeflag != tar.TypeReg || header.Size < 1 || header.Size > maxArchiveBytes {
			return fmt.Errorf("archive must contain one regular bounded meca executable")
		}
		file, err := directory.OpenFile("meca", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
		if err != nil {
			return err
		}
		_, err = io.Copy(file, reader)
		err = errors.Join(err, file.Sync(), file.Close())
		if err != nil {
			return err
		}
		found = true
	}
	// Finish gzip verification and bound padding or additional compressed members.
	if _, err := io.Copy(io.Discard, expanded); err != nil {
		return err
	}
	if expanded.N == 0 {
		return fmt.Errorf("expanded release archive exceeds 512 MiB")
	}
	if !found {
		return fmt.Errorf("release archive contains no meca executable")
	}
	return nil
}
