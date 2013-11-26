package plist

import "reflect"

type plistKind uint

const (
	Invalid plistKind = iota
	Dictionary
	Array
	String
	Integer
	Real
	Boolean
	Data
)

var plistKindNames map[plistKind]string = map[plistKind]string{
	Invalid:    "invalid",
	Dictionary: "dictionary",
	Array:      "array",
	String:     "string",
	Integer:    "integer",
	Real:       "real",
	Boolean:    "boolean",
	Data:       "data",
}

type plistValue struct {
	kind  plistKind
	value interface{}
}

type UnknownTypeError struct {
	Type reflect.Type
}

func (u *UnknownTypeError) Error() string {
	return "Unknown type " + u.Type.String()
}

