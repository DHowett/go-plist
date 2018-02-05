package plist

import (
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"runtime"
	"time"
)

type xmlPlistParser struct {
	xmlDecoder *xml.Decoder
}

func (p *xmlPlistParser) error(e string, args ...interface{}) {
	off := xmlInputOffset(p.xmlDecoder)
	panic(fmt.Errorf("%s at offset %v", fmt.Sprintf(e, args...), off))
}

func (p *xmlPlistParser) unexpected(token xml.Token) {
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
	for {
		if token, err := p.xmlDecoder.Token(); err == nil {
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
}

func (p *xmlPlistParser) next() xml.Token {
	token, err := p.xmlDecoder.Token()
	if err != nil {
		p.error("%v", err)
	}
	return token
}

func (p *xmlPlistParser) skip() {
	err := p.xmlDecoder.Skip()
	if err != nil {
		p.error("%v", err)
	}
}

func trimSpace(s string) string {
	b, e := 0, len(s)
	for ; b < e; b++ {
		if !whitespace.ContainsByte(s[b]) {
			break
		}
	}
	for ; e > b; e-- {
		if !whitespace.ContainsByte(s[e-1]) {
			break
		}
	}
	return s[b:e]
}

// opening tag has been consumed
func (p *xmlPlistParser) getNextString(element xml.StartElement) string {
	var s string
outer:
	for {
		token := p.next()
		switch token := token.(type) {
		case xml.EndElement:
			break outer
		case xml.CharData:
			s = string(token)
		default:
			p.unexpected(token)
		}
	}

	return trimSpace(s)
}

func (p *xmlPlistParser) mustGetNextString(element xml.StartElement) string {
	s := p.getNextString(element)
	if len(s) == 0 {
		p.error("empty <%s>", element.Name.Local)
	}
	return s
}

func (p *xmlPlistParser) parseStringElement(element xml.StartElement) cfString {
	return cfString(p.getNextString(element))
}

func (p *xmlPlistParser) parseIntegerElement(element xml.StartElement) *cfNumber {
	s := p.mustGetNextString(element)

	if s[0] == '-' {
		s, base := unsignedGetBase(s[1:])
		n := mustParseInt("-"+s, base, 64)
		return &cfNumber{signed: true, value: uint64(n)}
	}

	s, base := unsignedGetBase(s)
	n := mustParseUint(s, base, 64)
	return &cfNumber{signed: false, value: n}
}

func (p *xmlPlistParser) parseRealElement(element xml.StartElement) *cfReal {
	s := p.mustGetNextString(element)

	n := mustParseFloat(s, 64)
	return &cfReal{wide: true, value: n}

}

func (p *xmlPlistParser) parseDateElement(element xml.StartElement) cfDate {
	s := p.mustGetNextString(element)

	t, err := time.ParseInLocation(time.RFC3339, s, time.UTC)
	if err != nil {
		p.error("%v", err)
	}

	return cfDate(t)
}

func (p *xmlPlistParser) parseDataElement(element xml.StartElement) cfData {
	s := []byte(p.mustGetNextString(element))

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

func (p *xmlPlistParser) parseDictionary(element xml.StartElement) cfValue {
	keys := make([]string, 0, 32)
	values := make([]cfValue, 0, 32)
outer:
	for {
		token := p.next()

		switch token := token.(type) {
		case xml.StartElement:
			if token.Name.Local == "key" {
				keys = append(keys, p.getNextString(token))
			} else {
				if len(keys) != len(values)+1 {
					p.error("missing key in dictionary")
				}
				values = append(values, p.parseXMLElement(token))
			}
		case xml.EndElement:
			break outer
		case xml.CharData, xml.Comment:
			continue outer
		default:
			p.unexpected(token)
		}
	}
	return p.realizeKeysAndValues(keys, values)
}

func (p *xmlPlistParser) parseArray(element xml.StartElement) *cfArray {
	values := make([]cfValue, 0, 32)
outer:
	for {
		token := p.next()

		switch token := token.(type) {
		case xml.StartElement:
			values = append(values, p.parseXMLElement(token))
		case xml.EndElement:
			break outer
		case xml.CharData, xml.Comment:
			continue outer
		default:
			p.unexpected(token)
		}
	}
	return &cfArray{values}
}

func (p *xmlPlistParser) parseXMLElement(element xml.StartElement) cfValue {
	switch element.Name.Local {
	case "plist":
		// a <plist> should contain only one sub-element; we can safely recurse in here
	outer:
		for {
			token := p.next()
			switch token := token.(type) {
			case xml.EndElement:
				break outer
			case xml.StartElement:
				return p.parseXMLElement(token)
			case xml.CharData, xml.Comment:
				continue outer
			default:
				p.unexpected(token)
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
}

func newXMLPlistParser(r io.Reader) *xmlPlistParser {
	return &xmlPlistParser{xml.NewDecoder(r)}
}
