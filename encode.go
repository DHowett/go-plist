package plist

import (
	"encoding/base64"
	"encoding/xml"
	"io"
	"reflect"
)

const DOCTYPE = `DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"`

func (p *PlistEncoder) marshalPlistValueXML(plistVal *plistValue) error {
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
			p.marshalPlistValueXML(v)
		}
		p.xmlEncoder.EncodeToken(startElement.End())
	case Array:
		startElement := xml.StartElement{Name: xml.Name{Local: "array"}}
		p.xmlEncoder.EncodeToken(startElement)
		values := encodedValue.([]*plistValue)
		for _, v := range values {
			p.marshalPlistValueXML(v)
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

type PlistEncoder struct {
	writer     io.Writer
	xmlEncoder *xml.Encoder
}

func (p *PlistEncoder) EncodeDocument(v interface{}) error {
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
	err := p.Encode(v)
	p.xmlEncoder.EncodeToken(plistStartElement.End())
	p.xmlEncoder.Flush()
	return err
}

func (p *PlistEncoder) Encode(v interface{}) error {
	pv, err := valueToPlistValue(reflect.ValueOf(v))
	if err != nil {
		return err
	}
	return p.marshalPlistValueXML(pv)
}

func NewEncoder(w io.Writer) *PlistEncoder {
	p := &PlistEncoder{
		writer:     w,
		xmlEncoder: xml.NewEncoder(w),
	}
	return p
}

/*
func (p Plist) MarshalXML(encoder *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "plist"
	encoder.EncodeToken(start)
	marshalPlistObject(encoder, p.RootElement)
	encoder.EncodeToken(start.End())
	return nil
}
*/
