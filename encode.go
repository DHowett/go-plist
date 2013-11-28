// Package plist implements encoding and decoding of Apple's "property list" format.
// Property lists come in two sorts: XML and Binary. plist decodes both, but can only write XML.
// The mapping between property list and Go objects is described in the documentation for the Encode and Decode functions.
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

// An Encoder writes a property list to an output stream.
type Encoder struct {
	writer       io.Writer
	valueEncoder plistValueEncoder
}

// Encode writes the property list encoding of v to the connection.
//
// Encode traverses the value v recursively.
// Any nil values encountered, other than the root, will be silently discarded as
// the property list format bears no representation for nil values.
//
// Strings, integers of varying size, floats and booleans are encoded unchanged.
//
// Slice and Array values are encoded as property list arrays, except for
// []byte values, which are encoded as data.
//
// Map values encode as dictionaries. The map's key type must be string; there is no provision for encoding non-string dictionary keys.
//
// Struct values are encoded as dictionaries, with only exported fields being serialized. Struct field encoding may be influenced with the use of tags.
// The tag format is:
//
//     `plist:"<key>[,flags...]"`
//
// The following flags are supported:
//
//     omitempty    Only include the field if it is not set to the zero value for its type.
//
// If the key is "-", the field is ignored.
//
// Anonymous struct fields are encoded as if their exported fields were exposed via the outer struct.
//
// Pointer values encode as the value pointed to.
//
// Channel, complex and function values cannot be encoded. Any attempt to do so causes Encode to return an error.
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

// NewEncoder returns an Encoder that writes an XML property list to w
func NewEncoder(w io.Writer) *Encoder {
	p := &Encoder{
		valueEncoder: newXMLPlistValueEncoder(w),
	}
	return p
}
