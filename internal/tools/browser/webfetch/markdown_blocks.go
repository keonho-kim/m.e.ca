// Purpose: Preserve code blocks, lists and tables in extracted Markdown.
// Pattern: Block renderers sharing the document's bounded writer.
// Usage: markdown.visit dispatches supported HTML block nodes here.
// Related: internal/tools/browser/webfetch/markdown.go, internal/tools/browser/webfetch/content.go.
package webfetch

import (
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

func fenceLength(text string, char byte, minimum int) int {
	longest, length := minimum-1, 0
	for i := 0; i < len(text); i++ {
		if text[i] == char {
			length++
			longest = max(longest, length)
		} else {
			length = 0
		}
	}
	return longest + 1
}

func (m *markdown) preformatted(node *html.Node) {
	text := strings.TrimRight(codeText(node), "\r\n")
	backticks := fenceLength(text, '`', 3)
	fence := ""
	if backticks > 3 {
		tildes := fenceLength(text, '~', 3)
		if tildes < backticks {
			fence = strings.Repeat("~", tildes)
		}
	}
	if fence == "" {
		fence = strings.Repeat("`", backticks)
	}
	lang := ""
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Data != "code" || child.Type != html.ElementNode {
			continue
		}
		for field := range strings.FieldsSeq(attribute(child, "class")) {
			if name, found := strings.CutPrefix(field, "language-"); found && len(name) <= 32 && strings.IndexFunc(name, func(r rune) bool {
				return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' && r != '+' && r != '#'
			}) < 0 {
				lang = name
				break
			}
		}
		break
	}
	m.gap(2)
	m.write(fence)
	m.write(lang)
	m.write("\n")
	m.write(text)
	m.write("\n")
	m.write(fence)
	m.gap(2)
}

func (m *markdown) list(node *html.Node) {
	lines := 2
	if m.depth > 0 {
		lines = 1
	}
	m.gap(lines)
	index := 1
	if start, err := strconv.Atoi(attribute(node, "start")); err == nil && start > 0 && start < 1000000 {
		index = start
	}
	indent := strings.Repeat("  ", m.depth)
	m.depth++
	for child := node.FirstChild; child != nil && m.err == nil; child = child.NextSibling {
		if child.Type != html.ElementNode || child.Data != "li" || omitElement(child) {
			continue
		}
		m.gap(1)
		m.write(indent)
		if node.Data == "ol" {
			m.write(strconv.Itoa(index) + ". ")
			index++
		} else {
			m.write("- ")
		}
		for item := child.FirstChild; item != nil && m.err == nil; item = item.NextSibling {
			if item.Type == html.ElementNode && item.Data == "p" && !omitElement(item) {
				m.children(item)
			} else {
				m.visit(item)
			}
		}
	}
	m.depth--
	m.gap(lines)
}

func (m *markdown) table(node *html.Node) {
	m.gap(2)
	header := false
	for part := node.FirstChild; part != nil && m.err == nil; part = part.NextSibling {
		if omitElement(part) {
			continue
		}
		switch part.Data {
		case "tr":
			m.tableRow(part, &header)
		case "thead", "tbody", "tfoot":
			for row := part.FirstChild; row != nil && m.err == nil; row = row.NextSibling {
				if row.Type == html.ElementNode && row.Data == "tr" && !omitElement(row) {
					m.tableRow(row, &header)
				}
			}
		}
	}
	m.gap(2)
}

func (m *markdown) tableRow(row *html.Node, header *bool) {
	m.gap(1)
	columns := 0
	for cell := row.FirstChild; cell != nil && m.err == nil; cell = cell.NextSibling {
		if cell.Type != html.ElementNode || cell.Data != "td" && cell.Data != "th" || omitElement(cell) {
			continue
		}
		m.write("| ")
		m.inCell = true
		m.children(cell)
		m.inCell, m.space = false, false
		m.write(" ")
		columns++
	}
	if columns == 0 {
		return
	}
	m.write("|")
	if !*header {
		m.gap(1)
		for range columns {
			m.write("| --- ")
		}
		m.write("|")
		*header = true
	}
}
