package plist

import (
	"io"
	"reflect"
)

const DOCTYPE = `DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"`

type plistValueEncoder interface {
	encodeDocument(*plistValue) error
	encodePlistValue(*plistValue) error
}

type Encoder struct {
	writer       io.Writer
	valueEncoder plistValueEncoder
}

func (p *Encoder) EncodeDocument(v interface{}) error {
	pv, err := p.marshal(reflect.ValueOf(v))
	if err != nil {
		return err
	}
	return p.valueEncoder.encodeDocument(pv)
}

func (p *Encoder) Encode(v interface{}) error {
	pv, err := p.marshal(reflect.ValueOf(v))
	if err != nil {
		return err
	}
	return p.valueEncoder.encodePlistValue(pv)
}

func NewEncoder(w io.Writer) *Encoder {
	p := &Encoder{
		valueEncoder: newXMLPlistValueEncoder(w),
	}
	return p
}
