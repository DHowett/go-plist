package plist

import (
	"bytes"
	"io"
	"reflect"
	"runtime"
)

type parser interface {
	parseDocument() (*plistValue, error)
}

// A decoder reads a property list from an input stream.
type Decoder struct {
	reader io.ReadSeeker
	lax    bool
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
//
// As Go does not support 128-bit types, and we don't want to pretend we're giving the user integer types (as opposed to
// secretly passing them structs), we drop the high 64 bits of any 128-bit integers encoded in binary property lists.
//
// This is important because CoreFoundation serializes some large 64-bit values as 128-bit values with an empty high half.
func (p *Decoder) Decode(v interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()

	header := make([]byte, 6)
	p.reader.Read(header)
	p.reader.Seek(0, 0)

	var parser parser
	var pval *plistValue
	if bytes.Equal(header, []byte("bplist")) {
		parser = newBplistParser(p.reader)
		pval, err = parser.parseDocument()
		if err != nil {
			// Had a bplist header, but still got an error: we have to die here.
			panic(err)
		}
	} else {
		parser = newXMLPlistParser(p.reader)
		pval, err = parser.parseDocument()
		if err == io.EOF {
			parser = newTextPlistParser(p.reader)
			pval, err = parser.parseDocument()
			if err != nil {
				panic(err)
			}
		}
	}

	p.unmarshal(pval, reflect.ValueOf(v))
	return
}

// NewDecoder returns a Decoder that reads a property list from r.
// NewDecoder requires a Seekable stream as it reads 7 bytes
// from the start of r to determine the property list format,
// and then seeks back to the beginning of the stream.
func NewDecoder(r io.ReadSeeker) *Decoder {
	return &Decoder{reader: r, lax: false}
}
