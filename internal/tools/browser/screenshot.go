// Purpose: Read a bounded viewport image with one allocation and an explicit delivery reservation.
// Pattern: Session-owned file capture with caller-owned image lifetime.
// Usage: Session.Run returns a Page whose caller must Close after image decoding.
// Related: internal/tools/browser/results.go, internal/tools/local/browser.go.
package browser

import (
	"context"
	"image"
	_ "image/png"
	"io"
	"os"
	"path/filepath"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/processmemory"
)

func (s *Session) screenshot(ctx context.Context) (out Page, err error) {
	memory := processmemory.NewPayloadLease()
	defer func() {
		if err != nil {
			memory.Close()
		}
	}()
	if err = memory.Resize(16 << 20); err != nil {
		return Page{}, err
	}
	path := filepath.Join(s.work, "viewport.png")
	defer os.Remove(path)
	receipt, err := s.inspectWithURL(ctx, 32<<10, []string{"screenshot", path})
	receipt.Close()
	if err != nil {
		return Page{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return Page{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return Page{}, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 16<<20 {
		return Page{}, fault.Errorf(diag.BrowserLimit)
	}
	data, err := readViewport(file, info.Size())
	if err != nil {
		return Page{}, err
	}
	if err = ctx.Err(); err != nil {
		return Page{}, err
	}
	if err = memory.Resize(len(data)); err != nil {
		return Page{}, err
	}
	return Page{URL: s.redact(s.Current()), Text: diag.Text(diag.BrowserScreenshot), Untrusted: true, Image: data, memory: memory}, nil
}

func readViewport(file *os.File, size int64) ([]byte, error) {
	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return nil, err
	}
	if format != "png" || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > 2560*1600 {
		return nil, fault.Errorf(diag.BrowserLimit)
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	data := make([]byte, int(size))
	if _, err = io.ReadFull(file, data); err != nil {
		return nil, err
	}
	var extra [1]byte
	if n, readErr := file.Read(extra[:]); n != 0 || readErr != io.EOF {
		return nil, fault.Errorf(diag.BrowserLimit)
	}
	return data, nil
}
