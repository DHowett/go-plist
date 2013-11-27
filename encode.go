package plist

import (
	"errors"
	"io"
	"reflect"
	"runtime"
)

type plistValueEncoder interface {
	encodeDocument(*plistValue)
	encodePlistValue(*plistValue)
}

type Encoder struct {
	writer       io.Writer
	valueEncoder plistValueEncoder
}

func (p *Encoder) Encode(v interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()

	pval := p.marshal(reflect.ValueOf(v))
	if pval == nil {
		panic(errors.New("no root element to encode"))
	}

	p.valueEncoder.encodeDocument(pval)
	return
}

func NewEncoder(w io.Writer) *Encoder {
	p := &Encoder{
		valueEncoder: newXMLPlistValueEncoder(w),
	}
	return p
}
