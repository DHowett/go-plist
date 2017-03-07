package plist

import (
	"reflect"
)

// Property list format constants
const (
	// Used by Decoder to represent an invalid property list.
	InvalidFormat int = 0

	// Used to indicate total abandon with regards to Encoder's output format.
	AutomaticFormat = 0

	XMLFormat      = 1
	BinaryFormat   = 2
	OpenStepFormat = 3
	GNUStepFormat  = 4
)

var FormatNames = map[int]string{
	InvalidFormat:  "unknown/invalid",
	XMLFormat:      "XML",
	BinaryFormat:   "Binary",
	OpenStepFormat: "OpenStep",
	GNUStepFormat:  "GNUStep",
}

type unknownTypeError struct {
	typ reflect.Type
}

func (u *unknownTypeError) Error() string {
	return "plist: can't marshal value of type " + u.typ.String()
}

type invalidPlistError struct {
	format string
	err    error
}

func (e invalidPlistError) Error() string {
	s := "plist: invalid " + e.format + " property list"
	if e.err != nil {
		s += ": " + e.err.Error()
	}
	return s
}

type plistParseError struct {
	format string
	err    error
}

func (e plistParseError) Error() string {
	s := "plist: error parsing " + e.format + " property list"
	if e.err != nil {
		s += ": " + e.err.Error()
	}
	return s
}
