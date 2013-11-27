package plist

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"runtime"
)

type plistValueDecoder interface {
	decodeDocument() *plistValue
}

type Decoder struct {
	valueDecoder plistValueDecoder
}

func (p *Decoder) Decode(v interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()

	pval := p.valueDecoder.decodeDocument()
	p.unmarshal(pval, reflect.ValueOf(v))
	return
}

type noopDecoder struct{}

func (p *noopDecoder) decodeDocument() *plistValue {
	panic(errors.New("invalid property list document format"))
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
