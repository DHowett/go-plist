package plist

import (
	"encoding/base64"
	"encoding/xml"
	"errors"
	"io"
	"math"
	"strconv"
	"time"
)

const xmlDOCTYPE = `DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"`

type xmlPlistValueEncoder struct {
	writer     io.Writer
	xmlEncoder *xml.Encoder
}

func (p *xmlPlistValueEncoder) encodeDocument(pval *plistValue) error {
	p.writer.Write([]byte(xml.Header))
	p.xmlEncoder.EncodeToken(xml.Directive(xmlDOCTYPE))

	plistStartElement := xml.StartElement{
		Name: xml.Name{
			Local: "plist",
		},
		Attr: []xml.Attr{
			{
				Name: xml.Name{
					Local: "version",
				},
				Value: "1.0",
			},
		},
	}

	p.xmlEncoder.EncodeToken(plistStartElement)
	err := p.encodePlistValue(pval)
	p.xmlEncoder.EncodeToken(plistStartElement.End())
	p.xmlEncoder.Flush()
	return err
}

func (p *xmlPlistValueEncoder) encodePlistValue(pval *plistValue) error {
	defer p.xmlEncoder.Flush()

	key := ""
	encodedValue := pval.value
	switch pval.kind {
	case Dictionary:
		startElement := xml.StartElement{Name: xml.Name{Local: "dict"}}
		p.xmlEncoder.EncodeToken(startElement)
		values := encodedValue.(map[string]*plistValue)
		for k, v := range values {
			p.xmlEncoder.EncodeElement(k, xml.StartElement{Name: xml.Name{Local: "key"}})
			p.encodePlistValue(v)
		}
		p.xmlEncoder.EncodeToken(startElement.End())
	case Array:
		startElement := xml.StartElement{Name: xml.Name{Local: "array"}}
		p.xmlEncoder.EncodeToken(startElement)
		values := encodedValue.([]*plistValue)
		for _, v := range values {
			p.encodePlistValue(v)
		}
		p.xmlEncoder.EncodeToken(startElement.End())
	case String:
		key = "string"
	case Integer:
		key = "integer"
	case Real:
		key = "real"
		switch {
		case math.IsInf(pval.value.(float64), 1):
			encodedValue = "inf"
		case math.IsInf(pval.value.(float64), -1):
			encodedValue = "-inf"
		case math.IsNaN(pval.value.(float64)):
			encodedValue = "nan"
		}
	case Boolean:
		key = "false"
		b := pval.value.(bool)
		if b {
			key = "true"
		}
		encodedValue = ""
	case Data:
		key = "data"
		encodedValue = xml.CharData(base64.StdEncoding.EncodeToString(pval.value.([]byte)))
	case Date:
		key = "date"
		encodedValue = pval.value.(time.Time).Format(time.RFC3339)
	}
	if key != "" {
		return p.xmlEncoder.EncodeElement(encodedValue, xml.StartElement{Name: xml.Name{Local: key}})
	}
	return nil
}

func newXMLPlistValueEncoder(w io.Writer) *xmlPlistValueEncoder {
	return &xmlPlistValueEncoder{w, xml.NewEncoder(w)}
}

type xmlPlistValueDecoder struct {
	reader     io.Reader
	xmlDecoder *xml.Decoder
}

func (p *xmlPlistValueDecoder) decodeDocument() (*plistValue, error) {
	for {
		if token, err := p.xmlDecoder.Token(); err == nil {
			if element, ok := token.(xml.StartElement); ok {
				return p.decodeXMLElement(element)
			}
		} else {
			return nil, err
		}
	}
	return nil, nil
}

func (p *xmlPlistValueDecoder) decodeXMLElement(element xml.StartElement) (*plistValue, error) {
	var charData xml.CharData
	switch element.Name.Local {
	case "plist":
		for {
			token, err := p.xmlDecoder.Token()
			if err != nil {
				return nil, err
			}

			if el, ok := token.(xml.EndElement); ok && el.Name.Local == "plist" {
				break
			}

			if el, ok := token.(xml.StartElement); ok {
				return p.decodeXMLElement(el)
			}
		}
		return nil, nil
	case "string":
		err := p.xmlDecoder.DecodeElement(&charData, &element)
		if err != nil {
			return nil, err
		}

		return &plistValue{String, string(charData)}, nil
	case "integer":
		err := p.xmlDecoder.DecodeElement(&charData, &element)
		if err != nil {
			return nil, err
		}

		n, err := strconv.ParseUint(string(charData), 10, 64)
		if err != nil {
			return nil, err
		}

		return &plistValue{Integer, n}, nil
	case "real":
		err := p.xmlDecoder.DecodeElement(&charData, &element)
		if err != nil {
			return nil, err
		}

		n, err := strconv.ParseFloat(string(charData), 64)
		if err != nil {
			return nil, err
		}

		return &plistValue{Real, n}, nil
	case "true", "false":
		p.xmlDecoder.Skip()

		b := element.Name.Local == "true"
		return &plistValue{Boolean, b}, nil
	case "date":
		err := p.xmlDecoder.DecodeElement(&charData, &element)
		if err != nil {
			return nil, err
		}

		t, err := time.Parse(time.RFC3339, string(charData))
		if err != nil {
			return nil, err
		}

		return &plistValue{Date, t}, nil
	case "data":
		err := p.xmlDecoder.DecodeElement(&charData, &element)
		if err != nil {
			return nil, err
		}

		l := base64.StdEncoding.DecodedLen(len(charData))
		bytes := make([]uint8, l)
		l, err = base64.StdEncoding.Decode(bytes, charData)
		if err != nil {
			return nil, err
		}

		return &plistValue{Data, bytes[:l]}, nil
	case "dict":
		var key string
		var subvalues map[string]*plistValue = make(map[string]*plistValue)
		for {
			token, err := p.xmlDecoder.Token()
			if err != nil {
				return nil, err
			}

			if el, ok := token.(xml.EndElement); ok && el.Name.Local == "dict" {
				break
			}

			if el, ok := token.(xml.StartElement); ok {
				if el.Name.Local == "key" {
					p.xmlDecoder.DecodeElement(&key, &el)
				} else {
					if key == "" {
						return nil, errors.New("Missing key for value")
					}
					sval, err := p.decodeXMLElement(el)
					if err != nil {
						return nil, err
					}
					subvalues[key] = sval
				}
			}
		}
		return &plistValue{Dictionary, subvalues}, nil
	case "array":
		var subvalues []*plistValue = make([]*plistValue, 0, 10)
		for {
			token, err := p.xmlDecoder.Token()
			if err != nil {
				return nil, err
			}

			if el, ok := token.(xml.EndElement); ok && el.Name.Local == "array" {
				break
			}

			if el, ok := token.(xml.StartElement); ok {
				sval, err := p.decodeXMLElement(el)
				if err != nil {
					return nil, err
				}
				subvalues = append(subvalues, sval)
			}
		}
		return &plistValue{Array, subvalues}, nil
	}
	return nil, nil
}

func newXMLPlistValueDecoder(r io.Reader) *xmlPlistValueDecoder {
	return &xmlPlistValueDecoder{r, xml.NewDecoder(r)}
}
