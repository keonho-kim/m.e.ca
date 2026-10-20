// Purpose: Recognize sed script effects without mistaking patterns, replacement text or appended text for commands.
// Pattern: Bounded script scanner.
// Usage: The sed invocation parser applies returned write paths and embedded shell programs.
// Related: internal/policy/command/sed.go, internal/policy/command/sed_address.go.
package command

import "strings"

type sedEffects struct {
	writes, commands []string
	dynamic, unknown bool
}

type sedScanner struct {
	text    string
	index   int
	effects sedEffects
}

func sedProgram(text string) sedEffects {
	p := sedScanner{text: text}
	for p.index < len(p.text) {
		p.skip(" \t\r\n;}")
		if p.index == len(p.text) {
			break
		}
		if !p.addresses() {
			p.effects.unknown = true
			break
		}
		p.skip(" \t!")
		if p.index >= len(p.text) {
			p.effects.unknown = true
			break
		}
		action := p.text[p.index]
		p.index++
		switch action {
		case '#':
			p.restLine()
		case '{':
		case 's':
			p.substitute()
		case 'y':
			if p.index >= len(p.text) {
				p.effects.unknown = true
				break
			}
			delimiter := p.text[p.index]
			p.index++
			if !p.delimited(delimiter) || !p.delimited(delimiter) {
				p.effects.unknown = true
			}
		case 'w', 'W':
			if target := strings.TrimSpace(p.restLine()); target != "" {
				p.effects.writes = append(p.effects.writes, target)
			} else {
				p.effects.unknown = true
			}
		case 'e':
			if code := strings.TrimSpace(p.restLine()); code != "" {
				p.effects.commands = append(p.effects.commands, code)
			} else {
				p.effects.dynamic = true
			}
		case 'r', 'R':
			p.restLine()
		case 'a', 'i', 'c':
			p.literalText()
		case ':', 'b', 't', 'T':
			for p.index < len(p.text) && p.text[p.index] != ';' && p.text[p.index] != '\n' {
				p.index++
			}
		case 'q', 'Q', 'l':
			p.skip(" \t0123456789")
		case 'd', 'D', 'p', 'P', 'h', 'H', 'g', 'G', 'x', 'n', 'N', '=', 'z', 'F':
		case 'v':
			p.restLine()
		default:
			p.effects.unknown = true
		}
		if p.effects.unknown {
			break
		}
	}
	return p.effects
}

func (p *sedScanner) substitute() {
	if p.index >= len(p.text) {
		p.effects.unknown = true
		return
	}
	delimiter := p.text[p.index]
	p.index++
	if delimiter == '\n' || delimiter == '\\' || !p.delimited(delimiter) || !p.delimited(delimiter) {
		p.effects.unknown = true
		return
	}
	for p.index < len(p.text) {
		flag := p.text[p.index]
		if flag == ';' || flag == '\n' || flag == '}' {
			return
		}
		p.index++
		switch {
		case strings.ContainsRune("gpiImM \t\r", rune(flag)) || flag >= '0' && flag <= '9':
		case flag == 'w':
			if path := strings.TrimSpace(p.restLine()); path != "" {
				p.effects.writes = append(p.effects.writes, path)
			} else {
				p.effects.unknown = true
			}
			return
		case flag == 'e':
			p.effects.dynamic = true
		default:
			p.effects.unknown = true
			return
		}
	}
}

func (p *sedScanner) delimited(delimiter byte) bool {
	for p.index < len(p.text) {
		ch := p.text[p.index]
		p.index++
		if ch == '\\' {
			if p.index < len(p.text) {
				p.index++
			}
			continue
		}
		if ch == delimiter {
			return true
		}
	}
	return false
}

func (p *sedScanner) skip(characters string) {
	for p.index < len(p.text) && strings.ContainsRune(characters, rune(p.text[p.index])) {
		p.index++
	}
}

func (p *sedScanner) restLine() string {
	start := p.index
	for p.index < len(p.text) && p.text[p.index] != '\n' {
		p.index++
	}
	return p.text[start:p.index]
}

func (p *sedScanner) literalText() {
	p.skip(" \t")
	if p.index+1 < len(p.text) && p.text[p.index] == '\\' && p.text[p.index+1] == '\n' {
		p.index += 2
	}
	for p.index < len(p.text) {
		if p.text[p.index] == '\\' && p.index+1 < len(p.text) {
			p.index += 2
			continue
		}
		ch := p.text[p.index]
		p.index++
		if ch == '\n' {
			return
		}
	}
}
