package plist

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"runtime"
)

type generator interface {
	generateDocument(cfValue)
	Indent(string)
}

// An Encoder writes a property list to an output stream.
type Encoder struct {
	writer io.Writer
	format int

	options []Option
	indent  string
}

func (e *Encoder) unmarshalerSetLax(_ bool) (bool, error) {
	return false, nil
}

func (e *Encoder) generatorSetGNUStepBase64(_ bool) (bool, error) {
	return false, nil
}

func (e *Encoder) generatorSetIndent(_ string) (bool, error) {
	return false, nil
}

func (e *Encoder) encoderSetFormat(format int) (bool, error) {
	e.format = format
	return true, nil
}

// Encode writes the property list encoding of v to the stream.
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

	generatorOpts := make([]Option, 0, len(p.options))
	for _, opt := range p.options {
		// Apply options to ourself first (if possible)
		// Our option handlers don't throw errors
		applied, _ := opt(p)

		if !applied {
			generatorOpts = append(generatorOpts, opt)
		}
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
	g.Indent(p.indent)

	// Apply all options after format and indent (so the last one wins out if the user specifies
	// MarshalIndent(xxx, "\t", Indent("lol"))
	for _, opt := range generatorOpts {
		if receiver, ok := g.(optionReceiver); ok {
			_, err = opt(receiver)
			if err != nil {
				return
			}
		}
	}
	g.generateDocument(pval)
	return
}

// Indent turns on pretty-printing for the XML and Text property list formats.
// Each element begins on a new line and is preceded by one or more copies of indent according to its nesting depth.
func (p *Encoder) Indent(indent string) {
	p.indent = indent
}

func newEncoderWithOptions(w io.Writer, options ...Option) *Encoder {
	return &Encoder{
		writer:  w,
		format:  0,
		options: options,
	}
}

// NewEncoder returns an Encoder that writes an XML property list to w.
func NewEncoder(w io.Writer, options ...Option) *Encoder {
	return NewEncoderForFormat(w, XMLFormat, options...)
}

// NewEncoderForFormat returns an Encoder that writes a property list to w in the specified format.
// Pass AutomaticFormat to allow the library to choose the best encoding (currently BinaryFormat).
func NewEncoderForFormat(w io.Writer, format int, options ...Option) *Encoder {
	opts := make([]Option, 0, len(options)+1)
	opts = append(opts, Format(format))
	opts = append(opts, options...)
	return newEncoderWithOptions(w, opts...)
}

// NewBinaryEncoder returns an Encoder that writes a binary property list to w.
func NewBinaryEncoder(w io.Writer, options ...Option) *Encoder {
	return NewEncoderForFormat(w, BinaryFormat, options...)
}

// Marshal returns the property list encoding of v in the specified format.
//
// Pass AutomaticFormat to allow the library to choose the best encoding (currently BinaryFormat).
//
// Marshal traverses the value v recursively.
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
// Channel, complex and function values cannot be encoded. Any attempt to do so causes Marshal to return an error.
func Marshal(v interface{}, format int, options ...Option) ([]byte, error) {
	return MarshalIndent(v, format, "", options...)
}

// MarshalIndent works like Marshal, but each property list element
// begins on a new line and is preceded by one or more copies of indent according to its nesting depth.
func MarshalIndent(v interface{}, format int, indent string, options ...Option) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := NewEncoderForFormat(buf, format, options...)
	enc.Indent(indent)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
