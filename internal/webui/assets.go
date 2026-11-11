// Purpose: Serve immutable precompressed frontend assets directly from the executable.
// Pattern: Embedded HTTP adapter.
// Usage: Server startup constructs this handler only for --server --ui.
// Related: internal/webui/bundle.go, devtools/cmd/uiassets/bundle.go, internal/httpencoding/accept.go.
package webui

import (
	"compress/gzip"
	"encoding/json"
	"github.com/keonho-kim/meca/internal/httpencoding"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type Asset struct {
	File  string `json:"file"`
	Type  string `json:"type"`
	ETag  string `json:"etag"`
	Bytes int64  `json:"bytes"`
}
type Handler struct {
	files  fs.FS
	assets map[string]Asset
}

func New() (http.Handler, error) {
	files, err := bundledFiles()
	if err != nil {
		return nil, err
	}
	return newHandler(files)
}

func newHandler(files fs.FS) (*Handler, error) {
	manifest, err := fs.ReadFile(files, "manifest.json")
	if err != nil {
		return nil, fault.New(diag.FailedPrecondition, diag.UIBuildMissing)
	}
	if len(manifest) > 1<<20 {
		return nil, fault.New(diag.InvalidArgument, diag.UIBuildMissing)
	}
	var assets map[string]Asset
	if err := json.Unmarshal(manifest, &assets); err != nil {
		return nil, err
	}
	if _, ok := assets["index.html"]; !ok {
		return nil, fault.New(diag.FailedPrecondition, diag.UIBuildMissing)
	}
	for name, a := range assets {
		if !fs.ValidPath(name) || !fs.ValidPath(a.File) || a.Bytes < 0 {
			return nil, fault.New(diag.InvalidArgument, diag.UIBuildMissing)
		}
		if _, err := fs.Stat(files, a.File); err != nil {
			return nil, err
		}
	}
	return &Handler{files: files, assets: assets}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/ui/")
	if name == "" {
		name = "index.html"
	}
	a, ok := h.assets[name]
	if !ok && path.Ext(name) == "" && strings.Contains(r.Header.Get("Accept"), "text/html") {
		name = "index.html"
		a, ok = h.assets[name]
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' blob: data:; font-src 'self'; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Add("Vary", "Accept-Encoding")
	w.Header().Set("ETag", a.ETag)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	if name == "index.html" {
		w.Header().Set("Cache-Control", "no-cache")
	}
	compressed, identity := httpencoding.Accept(r.Header.Get("Accept-Encoding"))
	if !compressed && !identity {
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}
	for _, tag := range strings.Split(r.Header.Get("If-None-Match"), ",") {
		if tag = strings.TrimSpace(tag); tag == "*" || strings.TrimPrefix(tag, "W/") == strings.TrimPrefix(a.ETag, "W/") {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	typeName := a.Type
	if typeName == "" {
		typeName = mime.TypeByExtension(path.Ext(name))
	}
	w.Header().Set("Content-Type", typeName)
	f, err := h.files.Open(a.File)
	if err != nil {
		http.Error(w, diag.Text(diag.UIBuildMissing), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	if compressed {
		info, err := f.Stat()
		if err != nil {
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
		if r.Method != "HEAD" {
			_, _ = io.Copy(w, f)
		}
		return
	}
	w.Header().Set("Content-Length", strconv.FormatInt(a.Bytes, 10))
	if r.Method == "HEAD" {
		return
	}
	reader, err := gzip.NewReader(f)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer reader.Close()
	_, _ = io.Copy(w, reader)
}
