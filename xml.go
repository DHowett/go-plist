package plist

import (
	"encoding/base64"
	"encoding/xml"
	"io"
)

type xmlPlistValueEncoder struct {
	writer     io.Writer
	xmlEncoder *xml.Encoder
}

func (p *xmlPlistValueEncoder) encodeDocument(plistVal *plistValue) error {
	p.writer.Write([]byte(xml.Header))
	p.xmlEncoder.EncodeToken(xml.Directive(DOCTYPE))

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
	err := p.encodePlistValue(plistVal)
	p.xmlEncoder.EncodeToken(plistStartElement.End())
	p.xmlEncoder.Flush()
	return err
}

func (p *xmlPlistValueEncoder) encodePlistValue(plistVal *plistValue) error {
	defer p.xmlEncoder.Flush()

	key := ""
	encodedValue := plistVal.value
	switch plistVal.kind {
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
	case Boolean:
		key = "false"
		b := plistVal.value.(bool)
		if b {
			key = "true"
		}
		encodedValue = ""
	case Data:
		key = "data"
		encodedValue = xml.CharData(base64.StdEncoding.EncodeToString(plistVal.value.([]byte)))
	}
	if key != "" {
		return p.xmlEncoder.EncodeElement(encodedValue, xml.StartElement{Name: xml.Name{Local: key}})
	}
	return nil
}

func newXMLPlistValueEncoder(w io.Writer) *xmlPlistValueEncoder {
	return &xmlPlistValueEncoder{w, xml.NewEncoder(w)}
}
