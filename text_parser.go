package plist

import (
	"bufio"
	"encoding/hex"
	"errors"
	"io"
	"runtime"
	"strings"
	"time"
)

type byteReader interface {
	io.Reader
	io.ByteScanner
	Peek(n int) ([]byte, error)
	ReadBytes(delim byte) ([]byte, error)
}

type textPlistParser struct {
	reader             byteReader
	whitespaceReplacer *strings.Replacer
	format             int
}

func (p *textPlistParser) parseDocument() (pval cfValue, parseError error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			if _, ok := r.(invalidPlistError); ok {
				parseError = r.(error)
			} else {
				// Wrap all non-invalid-plist errors.
				parseError = plistParseError{"text", r.(error)}
			}
		}
	}()
	pval = p.parsePlistValue()
	return
}

func (p *textPlistParser) chugWhitespace() {
ws:
	for {
		c, err := p.reader.ReadByte()
		if err != nil && err != io.EOF {
			panic(err)
		}
		if whitespace[c/64]&(1<<(c%64)) == 0 {
			if c == '/' && err != io.EOF {
				// A / at the end of the file is not the begining of a comment.
				cs, err := p.reader.Peek(1)
				if err != nil && err != io.EOF {
					panic(err)
				}
				if err == io.EOF {
					return
				}
				c = cs[0]
				switch c {
				case '/':
					for {
						c, err = p.reader.ReadByte()
						if err != nil && err != io.EOF {
							panic(err)
						} else if err == io.EOF {
							break
						}
						// TODO: UTF-8
						if c == '\n' || c == '\r' {
							break
						}
					}
				case '*':
					// Peek returned a value here, so it is safe to read.
					_, _ = p.reader.ReadByte()
					star := false
					for {
						c, err = p.reader.ReadByte()
						if err != nil {
							panic(err)
						}
						if c == '*' {
							star = true
						} else if c == '/' && star {
							break
						} else {
							star = false
						}
					}
				default:
					p.reader.UnreadByte() // Not the beginning of a // or /* comment
					break ws
				}
				continue
			}
			p.reader.UnreadByte()
			break
		}
	}
}

func (p *textPlistParser) parseQuotedString() cfString {
	escaping := false
	s := ""
	for {
		byt, err := p.reader.ReadByte()
		// EOF here is an error: we're inside a quoted string!
		if err != nil {
			panic(err)
		}
		c := rune(byt)
		if !escaping {
			if c == '"' {
				break
			} else if c == '\\' {
				escaping = true
				continue
			}
		} else {
			escaping = false
			// Everything that is not listed here passes through unharmed.
			switch c {
			case 'a':
				c = '\a'
			case 'b':
				c = '\b'
			case 'v':
				c = '\v'
			case 'f':
				c = '\f'
			case 't':
				c = '\t'
			case 'r':
				c = '\r'
			case 'n':
				c = '\n'
			case 'x', 'u', 'U': // hex and unicode
				l := 4
				if c == 'x' {
					l = 2
				}
				hex := make([]byte, l)
				p.reader.Read(hex)
				newc := mustParseInt(string(hex), 16, 16)
				c = rune(newc)
			case '0', '1', '2', '3', '4', '5', '6', '7': // octal!
				oct := make([]byte, 3)
				oct[0] = uint8(c)
				p.reader.Read(oct[1:])
				newc := mustParseInt(string(oct), 8, 16)
				c = rune(newc)
			}
		}
		s += string(c)
	}
	return cfString(s)
}

func (p *textPlistParser) parseUnquotedString() cfString {
	s := ""
	for {
		c, err := p.reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		// if we encounter a character that must be quoted, we're done.
		// the GNUStep quote table is more lax here, so we use it instead of the OpenStep one.
		if gsQuotable[c/64]&(1<<(c%64)) > 0 {
			p.reader.UnreadByte()
			break
		}
		s += string(c)
	}

	if s == "" {
		panic(errors.New("invalid unquoted string (found an unquoted character that should be quoted?)"))
	}

	return cfString(s)
}

func (p *textPlistParser) parseDictionary() *cfDictionary {
	var keypv cfValue
	keys := make([]string, 0, 32)
	values := make([]cfValue, 0, 32)
	for {
		p.chugWhitespace()

		c, err := p.reader.ReadByte()
		// EOF here is an error: we're inside a dictionary!
		if err != nil {
			panic(err)
		}

		if c == '}' {
			break
		} else if c == '"' {
			keypv = p.parseQuotedString()
		} else {
			p.reader.UnreadByte() // Whoops, ate part of the string
			keypv = p.parseUnquotedString()
		}
		if keypv == nil {
			// TODO better error
			panic(errors.New("missing dictionary key"))
		}

		p.chugWhitespace()
		c, err = p.reader.ReadByte()
		if err != nil {
			panic(err)
		}

		if c != '=' {
			panic(errors.New("missing = in dictionary"))
		}

		// whitespace is guzzled within
		val := p.parsePlistValue()

		p.chugWhitespace()
		c, err = p.reader.ReadByte()
		if err != nil {
			panic(err)
		}

		if c != ';' {
			panic(errors.New("missing ; in dictionary"))
		}

		keys = append(keys, string(keypv.(cfString)))
		values = append(values, val)
	}

	return &cfDictionary{keys: keys, values: values}
}

func (p *textPlistParser) parseArray() *cfArray {
	values := make([]cfValue, 0, 32)
	for {
		c, err := p.reader.ReadByte()
		// EOF here is an error: we're inside an array!
		if err != nil {
			panic(err)
		}

		if c == ')' {
			break
		} else if c == ',' {
			continue
		}

		p.reader.UnreadByte()
		pval := p.parsePlistValue()
		if str, ok := pval.(cfString); ok && string(str) == "" {
			// Empty strings in arrays are apparently skipped?
			// TODO: Figure out why this was implemented.
			continue
		}
		values = append(values, pval)
	}
	return &cfArray{values}
}

func (p *textPlistParser) parseGNUStepValue(v []byte) cfValue {
	if len(v) < 3 {
		panic(errors.New("invalid GNUStep extended value"))
	}
	typ := v[1]
	v = v[2:]
	switch typ {
	case 'I':
		if v[0] == '-' {
			n := mustParseInt(string(v), 10, 64)
			return &cfNumber{signed: true, value: uint64(n)}
		} else {
			n := mustParseUint(string(v), 10, 64)
			return &cfNumber{signed: false, value: n}
		}
	case 'R':
		n := mustParseFloat(string(v), 64)
		return &cfReal{wide: true, value: n} // TODO(DH) 32/64
	case 'B':
		b := v[0] == 'Y'
		return cfBoolean(b)
	case 'D':
		t, err := time.Parse(textPlistTimeLayout, string(v))
		if err != nil {
			panic(err)
		}

		return cfDate(t.In(time.UTC))
	}
	panic(errors.New("invalid GNUStep type " + string(typ)))
}

func (p *textPlistParser) parsePlistValue() cfValue {
	for {
		p.chugWhitespace()

		c, err := p.reader.ReadByte()
		if err != nil && err != io.EOF {
			panic(err)
		}
		switch c {
		case '<':
			bytes, err := p.reader.ReadBytes('>')
			if err != nil {
				panic(err)
			}
			bytes = bytes[:len(bytes)-1]

			if len(bytes) == 0 {
				panic(errors.New("invalid empty angle-bracketed element"))
			}

			if bytes[0] == '*' {
				p.format = GNUStepFormat
				return p.parseGNUStepValue(bytes)
			} else {
				s := p.whitespaceReplacer.Replace(string(bytes))
				data, err := hex.DecodeString(s)
				if err != nil {
					panic(err)
				}
				return cfData(data)
			}
		case '"':
			return p.parseQuotedString()
		case '{':
			return p.parseDictionary()
		case '(':
			return p.parseArray()
		default:
			if gsQuotable[c/64]&(1<<(c%64)) > 0 {
				panic(errors.New("unexpected non-quotable character at root level"))
			}
			p.reader.UnreadByte() // Place back in buffer for parseUnquotedString
			return p.parseUnquotedString()
		}
	}
}

func newTextPlistParser(r io.Reader) *textPlistParser {
	var reader byteReader
	if rd, ok := r.(byteReader); ok {
		reader = rd
	} else {
		reader = bufio.NewReader(r)
	}
	return &textPlistParser{
		reader:             reader,
		whitespaceReplacer: strings.NewReplacer("\t", "", "\n", "", " ", "", "\r", ""),
		format:             OpenStepFormat,
	}
}
