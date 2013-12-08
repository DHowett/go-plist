package plist

import (
	"bufio"
	"encoding/hex"
	"errors"
	"io"
	"strconv"
	"time"
)

type textPlistGenerator struct {
	writer io.Writer
}

var (
	textPlistTimeLayout = "2006-01-02 15:04:05 -0700"
	padding             = "0000"
)

func (p *textPlistGenerator) generateDocument(pval *plistValue) {
	p.writePlistValue(pval)
}

func plistQuotedString(str string) string {
	s := ""
	quot := false
	for _, r := range str {
		if r > 0xFF {
			quot = true
			s += `\U`
			us := strconv.FormatInt(int64(r), 16)
			s += padding[len(us):]
			s += us
		} else if r > 0x7F {
			quot = true
			s += `\`
			us := strconv.FormatInt(int64(r), 8)
			s += padding[1+len(us):]
			s += us
		} else {
			c := uint8(r)
			if quotable[c/64]&(1<<(c%64)) > 0 {
				quot = true
			}

			switch c {
			case '\a':
				s += `\a`
			case '\b':
				s += `\b`
			case '\v':
				s += `\v`
			case '\f':
				s += `\f`
			case '\\':
				s += `\\`
			case '"':
				s += `\"`
			case '\t', '\r', '\n':
				fallthrough
			default:
				s += string(c)
			}
		}
	}
	if quot {
		s = `"` + s + `"`
	}
	return s
}

func (p *textPlistGenerator) writePlistValue(pval *plistValue) {
	if pval == nil {
		return
	}

	switch pval.kind {
	case Dictionary:
		p.writer.Write([]byte(`{`))
		dict := pval.value.(*dictionary)
		dict.populateArrays()
		for i, k := range dict.keys {
			io.WriteString(p.writer, plistQuotedString(k)+`=`)
			p.writePlistValue(dict.values[i])
			p.writer.Write([]byte(`;`))
		}
		p.writer.Write([]byte(`}`))
	case Array:
		p.writer.Write([]byte(`(`))
		values := pval.value.([]*plistValue)
		for _, v := range values {
			p.writePlistValue(v)
			p.writer.Write([]byte(`,`))
		}
		p.writer.Write([]byte(`)`))
	case String:
		io.WriteString(p.writer, plistQuotedString(pval.value.(string)))
	case Integer:
		if pval.value.(signedInt).signed {
			io.WriteString(p.writer, strconv.FormatInt(int64(pval.value.(signedInt).value), 10))
		} else {
			io.WriteString(p.writer, strconv.FormatUint(pval.value.(signedInt).value, 10))
		}
	case Real:
		io.WriteString(p.writer, strconv.FormatFloat(pval.value.(sizedFloat).value, 'g', -1, 64))
	case Boolean:
		b := pval.value.(bool)
		if b {
			p.writer.Write([]byte(`1`))
		} else {
			p.writer.Write([]byte(`0`))
		}
	case Data:
		b := pval.value.([]byte)
		hexencoded := make([]byte, hex.EncodedLen(len(b)))
		hex.Encode(hexencoded, b)
		io.WriteString(p.writer, `<`+string(hexencoded)+`>`)
	case Date:
		io.WriteString(p.writer, plistQuotedString(pval.value.(time.Time).In(time.UTC).Format(textPlistTimeLayout)))
	}
}

type byteReader interface {
	io.Reader
	io.ByteScanner
	ReadBytes(delim byte) ([]byte, error)
}

type textPlistParser struct {
	reader byteReader
}

func (p *textPlistParser) parseDocument() (*plistValue, error) {
	return p.parsePlistValue(), nil
}

func (p *textPlistParser) chugWhitespace() {
	for {
		c, err := p.reader.ReadByte()
		if err != nil && err != io.EOF {
			panic(err)
		}
		if whitespace[c/64]&(1<<(c%64)) == 0 {
			p.reader.UnreadByte()
			break
		}
	}
}

func (p *textPlistParser) parseQuotedString() *plistValue {
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
				newc, err := strconv.ParseInt(string(hex), 16, 16)
				if err != nil && err != io.EOF {
					panic(err)
				}
				c = rune(newc)
			case '0', '1', '2', '3', '4', '5', '6', '7': // octal!
				oct := make([]byte, 3)
				oct[0] = uint8(c)
				p.reader.Read(oct[1:])
				newc, err := strconv.ParseInt(string(oct), 8, 16)
				if err != nil && err != io.EOF {
					panic(err)
				}
				c = rune(newc)
			}
		}
		s += string(c)
	}
	return &plistValue{String, s}
}

func (p *textPlistParser) parseUnquotedString() *plistValue {
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
		if quotable[c/64]&(1<<(c%64)) > 0 {
			p.reader.UnreadByte()
			break
		}
		s += string(c)
	}
	return &plistValue{String, s}
}

func (p *textPlistParser) parseDictionary() *plistValue {
	var keypv *plistValue
	subval := make(map[string]*plistValue)
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
		if keypv == nil || keypv.value.(string) == "" {
			// TODO better error
			panic(errors.New("plist: missing dictionary key"))
		}

		p.chugWhitespace()
		c, err = p.reader.ReadByte()
		if err != nil {
			panic(err)
		}

		if c != '=' {
			panic(errors.New("plist: missing = in dictionary"))
		}

		// whitespace is guzzled within
		val := p.parsePlistValue()

		p.chugWhitespace()
		c, err = p.reader.ReadByte()
		if err != nil {
			panic(err)
		}

		if c != ';' {
			panic(errors.New("plist: missing ; in dictionary"))
		}

		subval[keypv.value.(string)] = val
	}
	return &plistValue{Dictionary, &dictionary{m: subval}}
}

func (p *textPlistParser) parseArray() *plistValue {
	subval := make([]*plistValue, 0, 10)
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
		subval = append(subval, p.parsePlistValue())
	}
	return &plistValue{Array, subval}
}

func (p *textPlistParser) parsePlistValue() *plistValue {
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
			data, err := hex.DecodeString(string(bytes))
			if err != nil {
				panic(err)
			}
			return &plistValue{Data, data}
		case '"':
			return p.parseQuotedString()
		case '{':
			return p.parseDictionary()
		case '(':
			return p.parseArray()
		default:
			p.reader.UnreadByte() // Place back in buffer for parseUnquotedString
			return p.parseUnquotedString()
		}
	}
	return nil
}

func newTextPlistParser(r io.Reader) *textPlistParser {
	var reader byteReader
	if rd, ok := r.(byteReader); ok {
		reader = rd
	} else {
		reader = bufio.NewReader(r)
	}
	return &textPlistParser{reader: reader}
}
