package plist

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"runtime"
	"time"
)

type xmlPlistParser struct {
	textBase
	reader   io.Reader
	tagStack []string
}

func (p *xmlPlistParser) mismatchedTags(start string, end string) {
	p.error("mismatched opening/closing tags <%s> and </%s>", start, end)
}

func (p *xmlPlistParser) unexpected(token string) {
	p.error("unexpected XML element `%v`", token)
}

func (p *xmlPlistParser) parseDocument() (pval cfValue, parseError error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			if _, ok := r.(invalidPlistError); ok {
				parseError = r.(error)
			} else {
				// Wrap all non-invalid-plist errors.
				parseError = plistParseError{"XML", r.(error)}
			}
		}
	}()
	buffer, err := ioutil.ReadAll(p.reader)
	if err != nil {
		panic(err)
	}

	p.input, err = guessEncodingAndConvert(buffer)
	if err != nil {
		panic(err)
	}

	p.skipWhitespace()
	pval = p.parseXMLElement()
	return
	/*
		for {
			if token, err := p.xmlDecoder.RawToken(); err == nil {
				if element, ok := token.(xml.StartElement); ok {
					pval = p.parseXMLElement(element)
					if pval == nil {
						panic(invalidPlistError{"XML", errors.New("no elements encountered")})
					}
					return
				}
			} else {
				// The first XML parse turned out to be invalid:
				// we do not have an XML property list.
				panic(invalidPlistError{"XML", err})
			}
		}
	*/
}

func (p *xmlPlistParser) skipWhitespace() {
	p.scanCharactersInSet(&whitespace)
	p.ignore()
}

var xmlTagSet = characterSet{
	0x5000800000000000, // < > /
	0x0000000000000000,
	0x0000000000000000,
	0x0000000000000000,
}

var tagCharacterSet = characterSet{
	0x07ff600000000000, // A-Za-z0-9_:.-
	0x07fffffe87fffffe,
	0x0000000000000000,
	0x0000000000000000,
}

func (p *xmlPlistParser) getNextString() string {
	p.skipWhitespace()
	p.scanCharactersNotInSet(&xmlTagSet)
	return p.emit()
}

// < has been consumed
func (p *xmlPlistParser) getTagName() string {
	p.ignore()
	p.scanCharactersInSet(&tagCharacterSet)
	return p.emit()
}

func (p *xmlPlistParser) parseStringElement() cfString {
	return cfString(p.getNextString())
}

func (p *xmlPlistParser) parseIntegerElement() *cfNumber {
	s := p.getNextString()
	if len(s) == 0 {
		p.error("empty <integer/>")
	}

	if s[0] == '-' {
		s, base := unsignedGetBase(s[1:])
		n := mustParseInt("-"+s, base, 64)
		return &cfNumber{signed: true, value: uint64(n)}
	}

	s, base := unsignedGetBase(s)
	n := mustParseUint(s, base, 64)
	return &cfNumber{signed: false, value: n}
}

func (p *xmlPlistParser) parseRealElement() *cfReal {
	s := p.getNextString()

	n := mustParseFloat(s, 64)
	return &cfReal{wide: true, value: n}

}

func (p *xmlPlistParser) parseDateElement() cfDate {
	s := p.getNextString()

	t, err := time.ParseInLocation(time.RFC3339, s, time.UTC)
	if err != nil {
		p.error("%v", err)
	}

	return cfDate(t)
}

func (p *xmlPlistParser) parseDataElement() cfData {
	s := []byte(p.getNextString())

	offset := 0
	for i, v := range s {
		if v != ' ' && v != '\t' && v != '\n' && v != '\r' {
			if offset != i {
				s[offset] = s[i]
			}
			offset++
		}
	}
	s = s[:offset]

	l := base64.StdEncoding.DecodedLen(offset)
	bytes := make([]uint8, l)

	var err error
	l, err = base64.StdEncoding.Decode(bytes, s)
	if err != nil {
		p.error("%v", err)
	}

	return cfData(bytes[:l])
}

func (p *xmlPlistParser) realizeKeysAndValues(keys []string, values []cfValue) cfValue {
	if len(keys) != len(values) {
		p.error("missing value in dictionary")
	}

	if len(keys) == 1 && keys[0] == "CF$UID" {
		if integer, ok := values[0].(*cfNumber); ok {
			return cfUID(integer.value)
		}
	}

	return &cfDictionary{keys: keys, values: values}
}

func (p *xmlPlistParser) parseDictionary() cfValue {
	return nil
	/*
			keys := make([]string, 0, 32)
			values := make([]cfValue, 0, 32)
		outer:
			for {
				token := p.next()

				switch token := token.(type) {
				case xml.EndElement:
					if token.Name.Local == "dict" {
						return p.realizeKeysAndValues(keys, values)
					} else {
						p.mismatchedTags(element, token)
					}
				case xml.StartElement:
					if token.Name.Local == "key" {
						k := p.getNextString(token)
						keys = append(keys, k)
					} else {
						if len(keys) != len(values)+1 {
							p.error("missing key in dictionary")
						}
						values = append(values, p.parseXMLElement(token))
					}
				case xml.Comment:
					continue outer // ignore all extraelemental data
				default:
					p.unexpected(token)
				}
			}
	*/
	return nil // shouldn't get here
}

func (p *xmlPlistParser) parseArray() *cfArray {
	values := make([]cfValue, 0, 32)
	/*
		outer:
			for {
				token := p.next()

				switch token := token.(type) {
				case xml.EndElement:
					if token.Name.Local == "array" {
						break outer
					}
					p.mismatchedTags(element, token)
				case xml.StartElement:
					values = append(values, p.parseXMLElement(token))
				case xml.Comment:
					continue outer // ignore all extraelemental data
				default:
					p.unexpected(token)
				}
			}
	*/
	return &cfArray{values}
}

func (p *xmlPlistParser) nextTag() (string, bool) {
	p.skipWhitespace()
	b := p.next()
	switch b {
	case '<':
		b = p.next()
		switch b {
		case '?', '!':
			p.scanUntil('>')
			p.pos++
			p.ignore()
			return "", false
		case '/':
			tag := p.getTagName()
			b = p.next()
			if b != '>' {
				p.error("unexpected '%c'", b)
			}
			if p.tagStack[len(p.tagStack)-1] != tag {
				p.mismatchedTags(p.tagStack[len(p.tagStack)-1], tag)
			}
			p.ignore()
			return tag, true
		default:
			p.backup()
		}
		p.skipWhitespace()
		tag := p.getTagName()
		p.skipWhitespace()

		// we don't care about attributes. just ignore them
		p.scanUntilAny("/>")
		empty := false
		b = p.next()
		switch b {
		case '/':
			empty = true
			b = p.next()
		}
		if b != '>' {
			p.error("unexpected '%c'", b)
		}
		_ = empty
		p.ignore()
		return tag, false
	}
	p.unexpected("non-tag?")
	return "", false
}

func (p *xmlPlistParser) parseXMLElement() cfValue {
	for {
		tag, close := p.nextTag()
		fmt.Println(tag, close)
		switch tag {
		case "plist":
			return p.parseXMLElement()
		case "string":
			return p.parseStringElement()
		case "integer":
			return p.parseIntegerElement()
		case "real":
			return p.parseRealElement()
		case "dict":
			return p.parseDictionary()
		case "array":
			return p.parseArray()
		case "true", "false": // small enough to inline
			b := tag == "true"
			return cfBoolean(b)
		case "":
			continue
		default:
			p.unexpected(tag)
		}
	}
	/*
		switch element.Name.Local {
		case "plist":
			// a <plist> should contain only one sub-element; we can safely recurse in here
			for {
				token := p.next()

				if el, ok := token.(xml.EndElement); ok && el.Name.Local == "plist" {
					break
				}

				if el, ok := token.(xml.StartElement); ok {
					return p.parseXMLElement(el)
				}
			}
			return nil
		case "string":
			return p.parseStringElement(element)
		case "integer":
			return p.parseIntegerElement(element)
		case "real":
			return p.parseRealElement(element)
		case "true", "false": // small enough to inline
			b := element.Name.Local == "true"
			p.skip() // skip the closing tag
			return cfBoolean(b)
		case "date":
			return p.parseDateElement(element)
		case "data":
			return p.parseDataElement(element)
		case "dict":
			return p.parseDictionary(element)
		case "array":
			return p.parseArray(element)
		default:
			p.unexpected(element)
			return nil
		}
	*/
	return nil
}

func newXMLPlistParser(r io.Reader) *xmlPlistParser {
	return &xmlPlistParser{reader: r}
}
