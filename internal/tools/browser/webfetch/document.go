// Purpose: Bound decoded web documents and own their temporary parsing memory.
// Pattern: Content-type dispatch over an admitted streaming reader.
// Usage: Session uses readDocument with a payload lease spanning redaction and paging.
// Related: internal/tools/browser/webfetch/client.go, internal/tools/browser/webfetch/markdown.go.
package webfetch

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/processmemory"
	"github.com/keonho-kim/meca/internal/tools/browser/chromium"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

const MaxDocumentBytes = 2 << 20
const maxTitleBytes = 256
const maxMarkup = 32 << 10

type document struct{ url, title, text string }

func (c *Client) readDocument(ctx context.Context, target *url.URL, lease *processmemory.PayloadLease, render chromium.Render) (result document, err error) {
	if err := lease.Resize(256 << 10); err != nil {
		return result, err
	}
	download, cancelDownload := context.WithTimeout(ctx, 15*time.Second)
	defer cancelDownload()
	opened, err := c.open(download, target, toolapi.WebFetchUserInput(ctx))
	if err != nil {
		return result, err
	}
	defer opened.connection.transport.CloseIdleConnections()
	response := opened.response
	defer func() {
		if response != nil {
			err = errors.Join(err, response.Body.Close())
		}
	}()
	if response.ContentLength > MaxDocumentBytes {
		return result, fault.New(diag.ResourceExhausted, diag.WebFetchBodyLimit)
	}
	contentType := response.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return result, fault.Errorf(diag.WebFetchMedia)
	}
	uncompressed := &io.LimitedReader{R: response.Body, N: MaxDocumentBytes + 1}
	var source io.Reader = uncompressed
	baseReservation := 256 << 10
	isHTML := mediaType == "text/html" || mediaType == "application/xhtml+xml"
	publicURL := *opened.address
	publicURL.RawQuery, publicURL.Fragment, publicURL.RawFragment, publicURL.ForceQuery = "", "", "", false
	switch {
	case isHTML:
		if render == nil {
			return result, fault.Errorf(diag.WebFetchBrowser)
		}
		if err := lease.Resize(3 * MaxDocumentBytes); err != nil {
			return result, err
		}
		body, readErr := readBody(response.Body, MaxDocumentBytes)
		address := opened.address.String()
		main := chromium.Response{Status: response.StatusCode, Header: response.Header, Body: body}
		closeErr := response.Body.Close()
		response = nil
		cancelDownload()
		if err := errors.Join(readErr, closeErr); err != nil {
			return result, err
		}
		if len(body) == 0 {
			return result, fault.Errorf(diag.WebFetchEmpty)
		}
		var markup string
		markup, err = render(ctx, address, main, opened.connection.loadResource)
		if err != nil {
			return result, err
		}
		baseReservation += 2 * len(markup)
		if err := lease.Resize(baseReservation); err != nil {
			return result, err
		}
		source = strings.NewReader(markup)
		uncompressed = nil
	case mediaType == "text/plain" || mediaType == "text/markdown":
		label := params["charset"]
		if label != "" && !strings.EqualFold(label, "utf-8") {
			source, err = charset.NewReaderLabel(label, uncompressed)
		}
	default:
		return result, fault.Errorf(diag.WebFetchMedia)
	}
	if err != nil {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		return result, fault.Errorf(diag.WebFetchExtraction)
	}
	reader := &documentReader{ctx: ctx, source: source, lease: lease, isHTML: isHTML, base: baseReservation, reserved: baseReservation}
	if isHTML {
		tree, parseErr := html.Parse(reader)
		if parseErr != nil {
			if reader.failure != nil {
				return result, reader.failure
			}
			return result, fault.Errorf(diag.WebFetchExtraction)
		}
		result.title, result.text, err = renderDocument(ctx, tree, &publicURL, func(outputBytes int) error {
			// Charge expanded links and escaped Markdown in addition to the HTML tree.
			return lease.Resize(reader.reserved + 3*outputBytes)
		})
	} else {
		var plain strings.Builder
		_, err = io.Copy(&plain, reader)
		result.text = plain.String()
	}
	if err != nil {
		return result, err
	}
	if uncompressed != nil && uncompressed.N == 0 {
		return result, fault.New(diag.ResourceExhausted, diag.WebFetchBodyLimit)
	}
	if !utf8.ValidString(result.title) || !utf8.ValidString(result.text) {
		return result, fault.Errorf(diag.WebFetchExtraction)
	}
	result.text = strings.TrimSpace(cleanText(result.text))
	if result.text == "" {
		return result, fault.Errorf(diag.WebFetchEmpty)
	}
	result.url = publicURL.String()
	return result, nil
}

type documentReader struct {
	ctx                     context.Context
	source                  io.Reader
	lease                   *processmemory.PayloadLease
	failure                 error
	bytes, markup, reserved int
	base                    int
	isHTML                  bool
}

func (r *documentReader) Read(data []byte) (n int, err error) {
	defer func() {
		if err != nil && err != io.EOF {
			r.failure = err
		}
	}()
	if err = r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err = r.source.Read(data[:min(len(data), MaxDocumentBytes+1-r.bytes)])
	r.bytes += n
	// The decoded-byte term covers text/attribute storage and bounded
	// serialization. Each markup candidate adds a tree allocation allowance.
	estimate := r.base + 16*r.bytes
	if r.isHTML {
		r.markup += bytes.Count(data[:n], []byte("<"))
		estimate += 256 * r.markup
	}
	if r.bytes > MaxDocumentBytes || r.markup > maxMarkup {
		return 0, fault.New(diag.ResourceExhausted, diag.WebFetchBodyLimit)
	}
	if estimate > r.reserved {
		const step = 256 << 10
		reserve := (estimate + step - 1) / step * step
		if resizeErr := r.lease.Resize(reserve); resizeErr != nil {
			return 0, resizeErr
		}
		r.reserved = reserve
	}
	return n, err
}

func cleanText(text string) string {
	unsafe := func(r rune) bool { return r < 0x20 && r != '\n' && r != '\t' && r != '\r' || r == 0x7f }
	if strings.IndexFunc(text, unsafe) < 0 {
		return text
	}
	return strings.Map(func(r rune) rune {
		if unsafe(r) {
			return -1
		}
		return r
	}, text)
}
