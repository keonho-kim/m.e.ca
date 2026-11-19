// Purpose: Render the selected document directly as bounded Markdown.
// Pattern: Single traversal with a whitespace-aware output writer.
// Usage: readDocument passes one HTML tree and the final response URL.
// Related: internal/tools/browser/webfetch/content.go, internal/tools/browser/webfetch/markdown_blocks.go.
package webfetch

import (
	"context"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"golang.org/x/net/html"
)

type markdown struct {
	ctx           context.Context
	base          *url.URL
	reserve       func(int) error
	body          strings.Builder
	lastHref      string
	lastLink      string
	err           error
	reserved      int
	pending       int
	visits, depth int
	space, inCell bool
}

const outputStep = 64 << 10

func renderDocument(ctx context.Context, root *html.Node, base *url.URL, reserve func(int) error) (string, string, error) {
	title, selected := selectContent(ctx, root)
	m := markdown{ctx: ctx, base: base, reserve: reserve, reserved: outputStep}
	m.visit(selected)
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	return title, m.body.String(), m.err
}

func (m *markdown) gap(lines int) {
	if m.inCell {
		m.space = true
		return
	}
	m.pending = max(m.pending, lines)
	m.space = false
}

func (m *markdown) write(text string) {
	if text == "" || m.err != nil {
		return
	}
	gap := m.pending
	space := m.space && gap == 0
	added := gap + len(text)
	if m.body.Len() == 0 {
		added = len(text)
	} else if space {
		added++
	}
	total := m.body.Len() + added
	if total > MaxDocumentBytes {
		m.err = fault.New(diag.ResourceExhausted, diag.WebFetchBodyLimit)
		return
	}
	if total > m.reserved {
		m.reserved = (total + outputStep - 1) / outputStep * outputStep
		if m.err = m.reserve(m.reserved); m.err != nil {
			return
		}
	}
	if m.body.Len() > 0 {
		for range gap {
			m.body.WriteByte('\n')
		}
		if space {
			m.body.WriteByte(' ')
		}
	}
	m.pending, m.space = 0, false
	m.body.WriteString(text)
}

func (m *markdown) text(value string) {
	start := 0
	for i, c := range value {
		switch {
		case unicode.IsSpace(c):
			m.write(value[start:i])
			m.space = true
			start = i + utf8.RuneLen(c)
		case c < 0x20 || c == 0x7f:
			m.write(value[start:i])
			start = i + 1
		case strings.ContainsRune("\\`*_[]|<>~", c):
			m.write(value[start:i])
			m.write("\\")
			start = i
		}
	}
	m.write(value[start:])
}

func (m *markdown) children(node *html.Node) {
	for child := node.FirstChild; child != nil && m.err == nil; child = child.NextSibling {
		m.visit(child)
	}
}

func (m *markdown) visit(node *html.Node) {
	if m.err != nil || omitElement(node) {
		return
	}
	m.visits++
	if m.visits%256 == 0 {
		if m.err = m.ctx.Err(); m.err != nil {
			return
		}
	}
	if node.Type == html.TextNode {
		m.text(node.Data)
		return
	}
	if node.Type != html.ElementNode {
		m.children(node)
		return
	}
	switch node.Data {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		m.gap(2)
		if !m.inCell {
			m.write(strings.Repeat("#", int(node.Data[1]-'0')) + " ")
		}
		m.children(node)
		m.gap(2)
	case "p", "div", "section", "article", "blockquote", "details", "summary", "dl", "dt", "dd":
		m.gap(2)
		m.children(node)
		m.gap(2)
	case "br":
		m.gap(1)
	case "pre":
		if m.inCell {
			m.text(codeText(node))
		} else {
			m.preformatted(node)
		}
	case "ul", "ol":
		if m.inCell {
			m.children(node)
		} else {
			m.list(node)
		}
	case "table":
		if m.inCell {
			m.children(node)
		} else {
			m.table(node)
		}
	case "li", "th", "td":
		m.gap(1)
		m.children(node)
		m.gap(1)
	case "strong", "b":
		m.emphasis(node, "**")
	case "em", "i":
		m.emphasis(node, "*")
	case "del", "s":
		m.emphasis(node, "~~")
	case "code":
		value := codeText(node)
		if strings.TrimSpace(value) != "" {
			fence := strings.Repeat("`", fenceLength(value, '`', 1))
			if m.inCell {
				value = strings.ReplaceAll(value, "|", "\\|")
				value = strings.ReplaceAll(value, "\n", " ")
			}
			m.write(fence)
			m.write(" ")
			m.write(value)
			m.write(" ")
			m.write(fence)
		}
	case "a":
		m.link(node)
	case "img":
		m.text(attribute(node, "alt"))
	default:
		m.children(node)
	}
}

func (m *markdown) emphasis(node *html.Node, marker string) {
	if node.FirstChild == nil {
		return
	}
	m.write(marker)
	m.children(node)
	m.space = false
	m.write(marker)
}

var markdownURL = strings.NewReplacer("(", "%28", ")", "%29", "|", "%7C")

func (m *markdown) link(node *html.Node) {
	address := attribute(node, "href")
	if len(address) == 0 || len(address) > MaxURLBytes || node.FirstChild == nil {
		m.children(node)
		return
	}
	// Reuse repeated targets with one retained resolution per document.
	if address != m.lastHref {
		m.lastHref, m.lastLink = address, ""
		target, err := m.base.Parse(address)
		if err == nil && target.User == nil && (target.Scheme == "https" || target.Scheme == "http") {
			m.lastLink = markdownURL.Replace(target.String())
		}
	}
	resolved := m.lastLink
	if resolved == "" {
		m.children(node)
		return
	}
	m.write("[")
	m.children(node)
	m.pending, m.space = 0, false
	m.write("](")
	m.write(resolved)
	m.write(")")
}
