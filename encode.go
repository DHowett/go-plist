package plist

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"runtime"
)

type generator interface {
	generateDocument(*plistValue)
}

// An Encoder writes a property list to an output stream.
type Encoder struct {
	writer io.Writer
	format int
}

// Encode writes the property list encoding of v to the connection.
//
// Encode traverses the value v recursively.
// Any nil values encountered, other than the root, will be silently discarded as
// the property list format bears no representation for nil values.
//
// Strings, integers of varying size, floats and booleans are encoded unchanged.
// Strings bearing non-ASCII runes will be encoded differently depending upon the property list format:
// UTF-8 for XML property lists and UTF-16 for binary property lists.
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
		panic(errors.New("plist: no root element to encode"))
	}

	var g generator
	switch p.format {
	case XMLFormat:
		g = newXMLPlistGenerator(p.writer)
	case BinaryFormat, AutomaticFormat:
		g = newBplistGenerator(p.writer)
	case OpenStepFormat, GNUStepFormat:
		g = newTextPlistGenerator(p.writer, p.format)
	}
	g.generateDocument(pval)
	return
}

// NewEncoder returns an Encoder that writes an XML property list to w.
func NewEncoder(w io.Writer) *Encoder {
	return NewEncoderForFormat(w, XMLFormat)
}

// NewEncoderForFormat returns an Encoder that writes a property list to w in the specified format.
// Pass AutomaticFormat to allow the library to choose the best encoding (currently BinaryFormat).
func NewEncoderForFormat(w io.Writer, format int) *Encoder {
	return &Encoder{
		writer: w,
		format: format,
	}
}

// NewBinaryEncoder returns an Encoder that writes a binary property list to w.
func NewBinaryEncoder(w io.Writer) *Encoder {
	return NewEncoderForFormat(w, BinaryFormat)
}

func Marshal(v interface{}, format int) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := NewEncoderForFormat(buf, format)
	err := enc.Encode(v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
