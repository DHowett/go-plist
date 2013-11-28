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

// A decoder reads a property list from an input stream.
type Decoder struct {
	valueDecoder plistValueDecoder
}

// Decode parses a property list document and stores the result in the value pointed to by v.
//
// Decode uses the inverse of the encodings that Encode uses, allocating heap-borne types as necessary.
//
// When given a nil pointer, Decode allocates a new value for it to point to.
//
// To decode property list values into an interface value, Decode decodes the property list into the concrete value contained
// in the interface value. If the interface value is nil, Decode stores one of the following in the interface value:
//
//     string, bool, uint64, float64
//     []byte, for plist data
//     []interface{}, for plist arrays
//     map[string]interface{}, for plist dictionaries
//
// If a property list value is not appropriate for a given value type, Decode aborts immediately and returns an error.
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

// NewDecoder returns a Decoder that reads a property list from r.
// NewDecoder reads 7 bytes from the start of r to determine the property list format.
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
