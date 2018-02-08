package plist

import (
	"encoding/base64"
	"encoding/xml"
	"io"
	"math"
	"time"

	"howett.net/plist/cf"
)

const xmlDOCTYPE = `<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
`

type xmlPlistGenerator struct {
	writer     io.Writer
	xmlEncoder *xml.Encoder
}

func (p *xmlPlistGenerator) generateDocument(root cf.Value) {
	io.WriteString(p.writer, xml.Header)
	io.WriteString(p.writer, xmlDOCTYPE)

	plistStartElement := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: "plist",
		},
		Attr: []xml.Attr{{
			Name: xml.Name{
				Space: "",
				Local: "version"},
			Value: "1.0"},
		},
	}

	p.xmlEncoder.EncodeToken(plistStartElement)

	p.writePlistValue(root)

	p.xmlEncoder.EncodeToken(plistStartElement.End())
	p.xmlEncoder.Flush()
}

func (p *xmlPlistGenerator) writeDictionary(dict *cf.Dictionary) {
	startElement := xml.StartElement{Name: xml.Name{Local: "dict"}}
	p.xmlEncoder.EncodeToken(startElement)
	dict.Range(func(i int, k string, v cf.Value) {
		p.xmlEncoder.EncodeElement(k, xml.StartElement{Name: xml.Name{Local: "key"}})
		p.writePlistValue(v)
	})
	p.xmlEncoder.EncodeToken(startElement.End())
}

func (p *xmlPlistGenerator) writeArray(a *cf.Array) {
	startElement := xml.StartElement{Name: xml.Name{Local: "array"}}
	p.xmlEncoder.EncodeToken(startElement)
	a.Range(func(i int, v cf.Value) {
		p.writePlistValue(v)
	})
	p.xmlEncoder.EncodeToken(startElement.End())
}

func (p *xmlPlistGenerator) writePlistValue(pval cf.Value) {
	if pval == nil {
		return
	}

	defer p.xmlEncoder.Flush()

	if dict, ok := pval.(*cf.Dictionary); ok {
		p.writeDictionary(dict)
		return
	} else if a, ok := pval.(*cf.Array); ok {
		p.writeArray(a)
		return
	} else if uid, ok := pval.(cf.UID); ok {
		p.writeDictionary(&cf.Dictionary{
			Keys: []string{"CF$UID"},
			Values: []cf.Value{
				&cf.Number{
					Signed: false,
					Value:  uint64(uid),
				},
			},
		})
		return
	}

	// Everything here and beyond is encoded the same way: <key>value</key>
	key := ""
	var encodedValue interface{} = pval

	switch pval := pval.(type) {
	case cf.String:
		key = "string"
	case *cf.Number:
		key = "integer"
		if pval.Signed {
			encodedValue = int64(pval.Value)
		} else {
			encodedValue = pval.Value
		}
	case *cf.Real:
		key = "real"
		encodedValue = pval.Value
		switch {
		case math.IsInf(pval.Value, 1):
			encodedValue = "inf"
		case math.IsInf(pval.Value, -1):
			encodedValue = "-inf"
		case math.IsNaN(pval.Value):
			encodedValue = "nan"
		}
	case cf.Boolean:
		key = "false"
		b := bool(pval)
		if b {
			key = "true"
		}
		encodedValue = ""
	case cf.Data:
		key = "data"
		encodedValue = xml.CharData(base64.StdEncoding.EncodeToString([]byte(pval)))
	case cf.Date:
		key = "date"
		encodedValue = time.Time(pval).In(time.UTC).Format(time.RFC3339)
	}

	if key != "" {
		err := p.xmlEncoder.EncodeElement(encodedValue, xml.StartElement{Name: xml.Name{Local: key}})
		if err != nil {
			panic(err)
		}
	}
}

func (p *xmlPlistGenerator) Indent(i string) {
	p.xmlEncoder.Indent("", i)
}

func newXMLPlistGenerator(w io.Writer) *xmlPlistGenerator {
	mw := mustWriter{w}
	return &xmlPlistGenerator{mw, xml.NewEncoder(mw)}
}
