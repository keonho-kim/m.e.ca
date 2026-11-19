// Purpose: Select readable HTML content and retain the document title in one tree pass.
// Pattern: Semantic subtree selection over the parsed HTML tree.
// Usage: renderDocument renders the selected subtree directly as Markdown.
// Related: internal/tools/browser/webfetch/document.go, internal/tools/browser/webfetch/markdown.go.
package webfetch

import (
	"context"
	"strings"

	"github.com/keonho-kim/meca/internal/tools/observation"
	"golang.org/x/net/html"
)

func selectContent(ctx context.Context, root *html.Node) (title string, selected *html.Node) {
	body := root
	for child := range root.Descendants() {
		if child.Type != html.ElementNode {
			continue
		}
		switch child.Data {
		case "title":
			if title == "" && child.FirstChild != nil {
				text := strings.TrimSpace(child.FirstChild.Data)
				title = strings.Clone(text[:observation.End(text, maxTitleBytes)])
			}
		case "body":
			body = child
		}
		if body != root {
			break
		}
	}
	var candidate *html.Node
	var candidateBytes, visits int
	var measure func(*html.Node) int
	measure = func(node *html.Node) int {
		visits++
		if visits%256 == 0 && ctx.Err() != nil || omitElement(node) {
			return 0
		}
		if node.Type == html.TextNode {
			return len(strings.TrimSpace(node.Data))
		}
		size := 0
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			size += measure(child)
		}
		if node.Type == html.ElementNode && (node.Data == "main" || node.Data == "article" || attribute(node, "role") == "main") && size >= candidateBytes {
			candidate, candidateBytes = node, size
		}
		return size
	}
	total := measure(body)
	if candidate != nil && candidateBytes > 0 && (candidateBytes >= 80 || candidateBytes >= total/4) {
		return title, candidate
	}
	return title, body
}

func omitElement(node *html.Node) bool {
	if node.Type != html.ElementNode {
		return false
	}
	switch node.Data {
	case "head", "script", "style", "template", "noscript", "nav", "aside", "footer", "form", "input", "button", "svg", "canvas", "iframe", "object":
		return true
	}
	for _, attr := range node.Attr {
		if attr.Key == "hidden" || attr.Key == "inert" || attr.Key == "aria-hidden" && strings.EqualFold(attr.Val, "true") {
			return true
		}
	}
	return false
}

func attribute(node *html.Node, name string) string {
	for _, attr := range node.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}
	return ""
}

func codeText(node *html.Node) string {
	if child := node.FirstChild; child != nil && child.NextSibling == nil && child.Type == html.TextNode {
		return child.Data
	}
	var text strings.Builder
	var appendText func(*html.Node)
	appendText = func(parent *html.Node) {
		for child := parent.FirstChild; child != nil; child = child.NextSibling {
			if omitElement(child) {
				continue
			}
			switch {
			case child.Type == html.TextNode:
				text.WriteString(child.Data)
			case child.Type == html.ElementNode && child.Data == "br":
				text.WriteByte('\n')
			default:
				appendText(child)
			}
		}
	}
	appendText(node)
	return text.String()
}
