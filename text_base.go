package plist

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type textBase struct {
	input string
	start int
	pos   int
	width int
}

func (p *textBase) error(e string, args ...interface{}) {
	line := strings.Count(p.input[:p.pos], "\n")
	char := p.pos - strings.LastIndex(p.input[:p.pos], "\n") - 1
	panic(fmt.Errorf("%s at line %d character %d", fmt.Sprintf(e, args...), line, char))
}

func (p *textBase) next() rune {
	if int(p.pos) >= len(p.input) {
		p.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(p.input[p.pos:])
	p.width = w
	p.pos += p.width
	return r
}

func (p *textBase) backup() {
	p.pos -= p.width
}

func (p *textBase) peek() rune {
	r := p.next()
	p.backup()
	return r
}

func (p *textBase) emit() string {
	s := p.input[p.start:p.pos]
	p.start = p.pos
	return s
}

func (p *textBase) ignore() {
	p.start = p.pos
}

func (p *textBase) empty() bool {
	return p.start == p.pos
}

func (p *textBase) scanUntil(ch rune) {
	if x := strings.IndexRune(p.input[p.pos:], ch); x >= 0 {
		p.pos += x
		return
	}
	p.pos = len(p.input)
}

func (p *textBase) scanUntilAny(chs string) {
	if x := strings.IndexAny(p.input[p.pos:], chs); x >= 0 {
		p.pos += x
		return
	}
	p.pos = len(p.input)
}

func (p *textBase) scanCharactersInSet(ch *characterSet) {
	for ch.Contains(p.next()) {
	}
	p.backup()
}

func (p *textBase) scanCharactersNotInSet(ch *characterSet) {
	var r rune
	for {
		r = p.next()
		if r == eof || ch.Contains(r) {
			break
		}
	}
	p.backup()
}
