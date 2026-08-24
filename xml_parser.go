package plist

import (
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type xmlPlistParser struct {
	reader             io.Reader
	xmlDecoder         *xml.Decoder
	whitespaceReplacer *strings.Replacer
	ntags              int
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
				if p.ntags == 0 {
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

func (p *xmlPlistParser) parseXMLElement(element xml.StartElement) cfValue {
	var charData xml.CharData
	switch element.Name.Local {
	case "plist":
		p.ntags++
		for {
			token, err := p.xmlDecoder.Token()
			if err != nil {
				panic(err)
			}

			if el, ok := token.(xml.EndElement); ok && el.Name.Local == "plist" {
				break
			}

			if el, ok := token.(xml.StartElement); ok {
				return p.parseXMLElement(el)
			}
		}
		return nil
	case "string":
		p.ntags++
		err := p.xmlDecoder.DecodeElement(&charData, &element)
		if err != nil {
			panic(err)
		}

		return cfString(charData)
	case "integer":
		p.ntags++
		err := p.xmlDecoder.DecodeElement(&charData, &element)
		if err != nil {
			panic(err)
		}

		s := string(charData)
		if len(s) == 0 {
			panic(errors.New("invalid empty <integer/>"))
		}

		if s[0] == '-' {
			s, base := unsignedGetBase(s[1:])
			n := mustParseInt("-"+s, base, 64)
			return &cfNumber{signed: true, value: uint64(n)}
		} else {
			s, base := unsignedGetBase(s)
			n := mustParseUint(s, base, 64)
			return &cfNumber{signed: false, value: n}
		}
	case "real":
		p.ntags++
		err := p.xmlDecoder.DecodeElement(&charData, &element)
		if err != nil {
			panic(err)
		}

		n := mustParseFloat(string(charData), 64)
		return &cfReal{wide: true, value: n}
	case "true", "false":
		p.ntags++
		p.xmlDecoder.Skip()

		b := element.Name.Local == "true"
		return cfBoolean(b)
	case "date":
		p.ntags++
		err := p.xmlDecoder.DecodeElement(&charData, &element)
		if err != nil {
			panic(err)
		}

		t, err := time.ParseInLocation(time.RFC3339, string(charData), time.UTC)
		if err != nil {
			panic(err)
		}

		return cfDate(t)
	case "data":
		p.ntags++
		err := p.xmlDecoder.DecodeElement(&charData, &element)
		if err != nil {
			panic(err)
		}

		str := p.whitespaceReplacer.Replace(string(charData))

		l := base64.StdEncoding.DecodedLen(len(str))
		bytes := make([]uint8, l)
		l, err = base64.StdEncoding.Decode(bytes, []byte(str))
		if err != nil {
			panic(err)
		}

		return cfData(bytes[:l])
	case "dict":
		p.ntags++
		var key *string
		keys := make([]string, 0, 32)
		values := make([]cfValue, 0, 32)
		for {
			token, err := p.xmlDecoder.Token()
			if err != nil {
				panic(err)
			}

			if el, ok := token.(xml.EndElement); ok && el.Name.Local == "dict" {
				if key != nil {
					panic(errors.New("missing value in dictionary"))
				}
				break
			}

			if el, ok := token.(xml.StartElement); ok {
				if el.Name.Local == "key" {
					var k string
					p.xmlDecoder.DecodeElement(&k, &el)
					key = &k
				} else {
					if key == nil {
						panic(errors.New("missing key in dictionary"))
					}
					keys = append(keys, *key)
					values = append(values, p.parseXMLElement(el))
					key = nil
				}
			}
		}

		dict := &cfDictionary{keys: keys, values: values}
		return dict.maybeUID(false)
	case "array":
		p.ntags++
		var key *int
		// Maintain a list of keys: either seen explicitly, or implicitly from
		// ordering in an array without keys.
		keys := make([]int, 0, 32)
		// Two flags to make note of what kind of array we have encountered so
		// far. Mixed type is currently not allowed.
		sawExplicitKey := false
		sawImplicitValue := false
		values := make([]cfValue, 0, 32)
		for {
			token, err := p.xmlDecoder.Token()
			if err != nil {
				panic(err)
			}

			if el, ok := token.(xml.EndElement); ok && el.Name.Local == "array" {
				break
			}

			if el, ok := token.(xml.StartElement); ok {
				if el.Name.Local == "key" {
					sawExplicitKey = true
					if sawImplicitValue {
						panic(errors.New("mixed type array"))
					}
					if key != nil {
						panic(errors.New("double key in array"))
					}
					var k int
					p.xmlDecoder.DecodeElement(&k, &el)
					key = &k
				} else {
					if key != nil {
						keys = append(keys, *key)
						key = nil
					} else {
						sawImplicitValue = true
						if sawExplicitKey {
							panic(errors.New("mixed type array"))
						}
						keys = append(keys, len(values))
					}
					values = append(values, p.parseXMLElement(el))
				}
			}
		}
		// If the array keys are non-continuous, return a dictionary.
		for i := 0; i < len(keys); i++ {
			if keys[i] != i {
				// ... but first convert the keys into strings.
				keys2 := make([]string, len(keys))
				for j, k := range keys {
					keys2[j] = strconv.Itoa(k)
				}
				dict := &cfDictionary{keys: keys2, values: values}
				return dict.maybeUID(false)
			}
		}
		// If the array is indeed continuous, return it as an array.
		return &cfArray{values}
	}
	err := fmt.Errorf("encountered unknown element %s", element.Name.Local)
	if p.ntags == 0 {
		// If out first XML tag is invalid, it might be an openstep data element, ala <abab> or <0101>
		panic(invalidPlistError{"XML", err})
	}
	panic(err)
}

func newXMLPlistParser(r io.Reader) *xmlPlistParser {
	return &xmlPlistParser{r, xml.NewDecoder(r), strings.NewReplacer("\t", "", "\n", "", " ", "", "\r", ""), 0}
}
