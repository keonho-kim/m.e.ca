// Purpose: Skip sed numeric and regular-expression addresses before identifying actual script commands.
// Pattern: Script-scanner subroutine.
// Usage: sedProgram calls addresses for each statement.
// Related: internal/policy/command/sed_program.go.
package command

func (p *sedScanner) addresses() bool {
	for count := 0; count < 2; count++ {
		p.skip(" \t")
		if p.index >= len(p.text) {
			return true
		}
		ch := p.text[p.index]
		switch {
		case ch >= '0' && ch <= '9' || ch == '$':
			p.index++
			p.skip("0123456789~")
		case ch == '/':
			p.index++
			if !p.delimited('/') {
				return false
			}
		case ch == '\\' && p.index+1 < len(p.text):
			delimiter := p.text[p.index+1]
			p.index += 2
			if !p.delimited(delimiter) {
				return false
			}
		case count == 1 && (ch == '+' || ch == '~'):
			p.index++
			p.skip("0123456789")
		default:
			return true
		}
		p.skip(" \t")
		if p.index < len(p.text) && p.text[p.index] == ',' {
			p.index++
			continue
		}
		return true
	}
	return true
}
