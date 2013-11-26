package plist

import (
	"io"
	"reflect"
)

type plistValueEncoder interface {
	encodeDocument(*plistValue) error
	encodePlistValue(*plistValue) error
}

type Encoder struct {
	writer       io.Writer
	valueEncoder plistValueEncoder
}

func (p *Encoder) Encode(v interface{}) error {
	pv, err := p.marshal(reflect.ValueOf(v))
	if err != nil {
		return err
	}
	return p.valueEncoder.encodeDocument(pv)
}

func NewEncoder(w io.Writer) *Encoder {
	p := &Encoder{
		valueEncoder: newXMLPlistValueEncoder(w),
	}
	return p
}
