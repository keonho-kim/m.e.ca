// Purpose: Read regular skill files through a pinned directory and enforce access limits.
// Pattern: Filesystem I/O boundary.
// Usage: Discovery, loading and resource reads share openRoot and openRegular.
// Related: internal/skills/reader.go, internal/skills/resources.go.
package skills

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"unicode/utf8"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func (r *Reader) openRoot(ctx context.Context, path string) (*os.Root, string, error) {
	if r.check == nil || r.redact == nil {
		return nil, "", faults.Errorf(diagcatalog.SkillsSkillAccessBoundaryIsRequired)
	}
	actual, err := filepath.EvalSymlinks(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	if err := r.check(actual); err != nil {
		return nil, "", err
	}
	before, err := os.Stat(actual)
	if err != nil {
		return nil, "", err
	}
	var root *os.Root
	if confined, ok := ctx.Value(readRootKey{}).(*os.Root); ok && confined != nil {
		var relative string
		relative, err = filepath.Rel(confined.Name(), actual)
		if err != nil || !filepath.IsLocal(relative) {
			return nil, "", faults.Errorf(diagcatalog.SkillsSkillPathIsOutsideProjectScope, ErrInvalid)
		}
		root, err = confined.OpenRoot(relative)
	} else {
		root, err = os.OpenRoot(actual)
	}
	if err != nil {
		return nil, "", err
	}
	after, err := root.Stat(".")
	if err != nil || !os.SameFile(before, after) {
		return nil, "", errors.Join(faults.Errorf(diagcatalog.SkillsSkillDirectoryChangedDuringOpen), err, root.Close())
	}
	return root, actual, nil
}

func openRegular(root *os.Root, name string, limit int64) (*os.File, error) {
	file, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > limit {
		return nil, errors.Join(faults.Errorf(diagcatalog.SkillsARegularFileWithinTheSizeLimit, ErrLimit), err, file.Close())
	}
	return file, nil
}

func (r *Reader) document(ctx context.Context, path, scope string, metadataOnly bool) (doc instruction.Document, actual string, err error) {
	if err = ctx.Err(); err != nil {
		return
	}
	root, actual, err := r.openRoot(ctx, path)
	if err != nil {
		return doc, actual, err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	doc, err = r.documentAt(ctx, root, path, scope, actual, metadataOnly)
	return doc, actual, err
}

func (r *Reader) documentAt(ctx context.Context, root *os.Root, path, scope, actual string, metadataOnly bool) (doc instruction.Document, err error) {
	file, err := openRegular(root, "SKILL.md", MaxDocumentBytes)
	if err != nil {
		return doc, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	before, err := file.Stat()
	if err != nil {
		return doc, err
	}
	var body []byte
	if metadataOnly {
		body, err = readMetadata(ctx, file)
	} else {
		body, err = io.ReadAll(io.LimitReader(file, MaxDocumentBytes+1))
	}
	if err != nil {
		return doc, err
	}
	after, err := root.Stat("SKILL.md")
	if err != nil {
		return doc, err
	}
	if !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return doc, faults.Errorf(diagcatalog.SkillsSKILLMdChangedDuringRead, ErrInvalid)
	}
	meta, start, err := frontmatter(body[:min(len(body), MaxMetadataBytes)])
	if err != nil {
		return doc, err
	}
	doc.Metadata, err = parseMetadata(meta, filepath.Base(actual))
	if err != nil {
		return doc, err
	}
	doc.ID, doc.Source = scope+":"+filepath.Base(path), scope
	if scope == "path" {
		doc.ID = fmt.Sprintf("path:%x", sha256.Sum256([]byte(actual)))
	}
	doc.Description = r.clean(doc.Description)
	if r.clean(doc.ID) != doc.ID || r.clean(doc.Name) != doc.Name {
		return doc, faults.Errorf(diagcatalog.SkillsProtectedCredentialInSkillIdentity, ErrInvalid)
	}
	if err = readInvocationPolicy(root, &doc.Metadata); err != nil {
		return doc, err
	}
	if !metadataOnly {
		if len(body) > MaxDocumentBytes {
			return doc, ErrLimit
		}
		if !utf8.Valid(body) {
			return doc, faults.Errorf(diagcatalog.SkillsSKILLMdMustBeUTF, ErrInvalid)
		}
		doc.Body = r.clean(string(body[start:]))
		metadata, encodeErr := json.Marshal(doc.Metadata)
		if encodeErr != nil {
			return doc, encodeErr
		}
		digest := sha256.New()
		_, _ = digest.Write(metadata)
		_, _ = digest.Write([]byte{0})
		_, _ = io.WriteString(digest, doc.Body)
		doc.Revision = fmt.Sprintf("%x", digest.Sum(nil))
		if len(doc.Body) > MaxDocumentBytes {
			return doc, ErrLimit
		}
	}
	return doc, ctx.Err()
}

func readMetadata(ctx context.Context, file io.Reader) ([]byte, error) {
	reader := bufio.NewReaderSize(io.LimitReader(file, MaxMetadataBytes+1), 1024)
	var body []byte
	first := true
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		line, err := reader.ReadBytes('\n')
		if len(body)+len(line) > MaxMetadataBytes {
			return nil, ErrLimit
		}
		body = append(body, line...)
		marker := bytes.TrimSuffix(bytes.TrimSuffix(line, []byte("\n")), []byte("\r"))
		if first {
			if string(marker) != "---" {
				return nil, faults.Errorf(diagcatalog.SkillsSKILLMdRequiresYAMLFrontmatter, ErrInvalid)
			}
			first = false
		} else if string(marker) == "---" {
			return body, nil
		}
		if err != nil {
			return nil, faults.Errorf(diagcatalog.SkillsMissingFrontmatterTerminator, ErrInvalid)
		}
	}
}
