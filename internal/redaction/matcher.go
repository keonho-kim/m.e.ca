// Purpose: Index immutable credential prefixes without scanning every remembered key per character.
// Pattern: Compressed prefix tree shared by response-owned stream masks.
// Usage: Remember publishes a new matcher; NewStreamMask borrows it under the vault read lock.
// Related: internal/redaction/redaction.go, internal/redaction/stream.go.
package redaction

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type keyNode struct {
	prefix   string
	children []keyNode
	terminal bool
}

type keyIndex struct{ starts [256]*keyNode }

func indexKeys(keys []string) *keyIndex {
	ordered := append([]string(nil), keys...)
	sort.Strings(ordered)
	index := &keyIndex{}
	for first := 0; first < len(ordered); {
		end := first + 1
		for end < len(ordered) && ordered[end][0] == ordered[first][0] {
			end++
		}
		node := buildKeyNode(ordered[first:end], 0)
		index.starts[ordered[first][0]] = &node
		first = end
	}
	return index
}

func buildKeyNode(keys []string, start int) keyNode {
	first, last := keys[0], keys[len(keys)-1]
	end := start
	for end < min(len(first), len(last)) && first[end] == last[end] {
		end++
	}
	node := keyNode{prefix: first[start:end], terminal: end == len(first)}
	if node.terminal {
		keys = keys[1:]
	}
	for at := 0; at < len(keys); {
		next := at + 1
		for next < len(keys) && keys[next][end] == keys[at][end] {
			next++
		}
		node.children = append(node.children, buildKeyNode(keys[at:next], end))
		at = next
	}
	return node
}

// match returns the longest complete original range and whether another key
// still needs more input. Controls may be removed between credential bytes.
func (index *keyIndex) match(text string) (matched int, partial bool) {
	if index == nil || len(text) == 0 {
		return 0, false
	}
	node := index.starts[text[0]]
	at := 0
	for node != nil {
		if strings.HasPrefix(text[at:], node.prefix) {
			at += len(node.prefix)
		} else {
			for i := 0; i < len(node.prefix); i++ {
				var pending bool
				at, pending = skipKeyControls(text, at)
				if pending || at == len(text) {
					return matched, true
				}
				if text[at] != node.prefix[i] {
					return matched, false
				}
				at++
			}
		}
		if node.terminal {
			matched = at
		}
		if len(node.children) == 0 {
			return matched, false
		}
		var pending bool
		at, pending = skipKeyControls(text, at)
		if pending || at == len(text) {
			return matched, true
		}
		children := node.children
		position := sort.Search(len(children), func(i int) bool { return children[i].prefix[0] >= text[at] })
		if position == len(children) || children[position].prefix[0] != text[at] {
			return matched, false
		}
		node = &children[position]
	}
	return matched, false
}

func skipKeyControls(text string, at int) (int, bool) {
	for at < len(text) {
		b := text[at]
		if b >= ' ' && b < 127 || b == '\n' || b == '\t' || !utf8.RuneStart(b) {
			return at, false
		}
		if !utf8.FullRuneInString(text[at:]) {
			return at, true
		}
		r, size := utf8.DecodeRuneInString(text[at:])
		if !unicode.IsControl(r) {
			return at, false
		}
		at += size
	}
	return at, false
}
