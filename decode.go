package plist

import (
	"bytes"
	"errors"
	"io"
	"reflect"
)

type plistValueDecoder interface {
	decodeDocument() (*plistValue, error)
}

type Decoder struct {
	valueDecoder plistValueDecoder
}

func (p *Decoder) Decode(v interface{}) error {
	pval, err := p.valueDecoder.decodeDocument()
	if err != nil {
		return err
	}

	return p.unmarshal(pval, reflect.ValueOf(v))
}

type noopDecoder struct{}

func (p *noopDecoder) decodeDocument() (*plistValue, error) {
	return nil, errors.New("invalid property list document format")
}

func NewDecoder(r io.ReadSeeker) *Decoder {
	header := make([]byte, 7)
	r.Read(header)
	r.Seek(0, 0)

	var decoder plistValueDecoder

	if bytes.Equal(header, []byte("bplist0")) {
		decoder = newBplistValueDecoder(r)
	} else if bytes.Contains(header, []byte("<")) {
		decoder = newXMLPlistValueDecoder(r)
	} else {
		decoder = &noopDecoder{}
	}
	return &Decoder{decoder}
}
