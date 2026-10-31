// Purpose: Keep large head-and-tail command captures in bounded executor-owned files.
// Pattern: Lazy spill buffer with fixed disk regions.
// Usage: Captures stay in memory for small output and spill after one MiB.
// Related: internal/tools/local/process/output.go, internal/tools/local/executor.go.
package process

import (
	"errors"
	"io"
	"os"
	"strings"
)

const memoryCaptureBytes = 1024 * 1024

type spillCapture struct {
	file              *captureFile
	head, tail, start int
	truncated         bool
}

func openSpill(directory string, head []byte) (*spillCapture, error) {
	file, err := newCaptureFile(directory)
	if err != nil {
		return nil, err
	}
	s := &spillCapture{file: file}
	if err = s.write(head); err != nil {
		return nil, errors.Join(err, s.close())
	}
	return s, nil
}

func (s *spillCapture) write(content []byte) error {
	n := min(len(content), captureHalf-s.head)
	if n > 0 {
		if _, err := s.file.WriteAt(content[:n], int64(s.head)); err != nil {
			return err
		}
		s.head += n
		content = content[n:]
	}
	if len(content) >= captureHalf {
		s.truncated = s.truncated || len(content) > captureHalf || s.tail > 0
		_, err := s.file.WriteAt(content[len(content)-captureHalf:], captureHalf)
		s.tail, s.start = captureHalf, 0
		return err
	}
	n = min(len(content), captureHalf-s.tail)
	if n > 0 {
		if _, err := s.file.WriteAt(content[:n], int64(captureHalf+s.tail)); err != nil {
			return err
		}
		s.tail += n
		content = content[n:]
	}
	if len(content) == 0 {
		return nil
	}
	s.truncated = true
	n = min(len(content), captureHalf-s.start)
	if _, err := s.file.WriteAt(content[:n], int64(captureHalf+s.start)); err != nil {
		return err
	}
	if _, err := s.file.WriteAt(content[n:], captureHalf); err != nil {
		return err
	}
	s.start = (s.start + len(content)) % captureHalf
	return nil
}

func (s *spillCapture) snapshot() (string, error) {
	var out strings.Builder
	out.Grow(s.head + s.tail + 32)
	copySection := func(at, count int) error {
		if count == 0 {
			return nil
		}
		_, err := io.CopyN(&out, io.NewSectionReader(s.file, int64(at), int64(count)), int64(count))
		return err
	}
	if err := copySection(0, s.head); err != nil {
		return "", err
	}
	if s.truncated {
		out.WriteString("\n[... output omitted ...]\n")
	}
	if err := copySection(captureHalf+s.start, s.tail-s.start); err != nil {
		return "", err
	}
	if err := copySection(captureHalf, s.start); err != nil {
		return "", err
	}
	return out.String(), nil
}

func (s *spillCapture) close() error { return errors.Join(s.file.Close(), os.Remove(s.file.Name())) }
