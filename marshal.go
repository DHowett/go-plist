package plist

import (
	"encoding"
	"reflect"
	"time"

	"howett.net/plist/cf"
)

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

var (
	plistMarshalerType = reflect.TypeOf((*Marshaler)(nil)).Elem()
	textMarshalerType  = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	timeType           = reflect.TypeOf((*time.Time)(nil)).Elem()
)

func implementsInterface(val reflect.Value, interfaceType reflect.Type) (interface{}, bool) {
	if val.CanInterface() && val.Type().Implements(interfaceType) {
		return val.Interface(), true
	}

	if val.CanAddr() {
		pv := val.Addr()
		if pv.CanInterface() && pv.Type().Implements(interfaceType) {
			return pv.Interface(), true
		}
	}
	return nil, false
}

func (p *Encoder) marshalPlistInterface(marshalable Marshaler) cf.Value {
	value, err := marshalable.MarshalPlist()
	if err != nil {
		panic(err)
	}
	return p.marshal(reflect.ValueOf(value))
}

// marshalTextInterface marshals a TextMarshaler to a plist string.
func (p *Encoder) marshalTextInterface(marshalable encoding.TextMarshaler) cf.Value {
	s, err := marshalable.MarshalText()
	if err != nil {
		panic(err)
	}
	return cf.String(s)
}

// marshalStruct marshals a reflected struct value to a plist dictionary
func (p *Encoder) marshalStruct(typ reflect.Type, val reflect.Value) cf.Value {
	tinfo, _ := getTypeInfo(typ)

	dict := &cf.Dictionary{
		Keys:   make([]string, 0, len(tinfo.fields)),
		Values: make([]cf.Value, 0, len(tinfo.fields)),
	}
	for _, finfo := range tinfo.fields {
		value := finfo.value(val)
		if !value.IsValid() || finfo.omitEmpty && isEmptyValue(value) {
			continue
		}
		dict.Keys = append(dict.Keys, finfo.name)
		dict.Values = append(dict.Values, p.marshal(value))
	}

	return dict
}

func (p *Encoder) marshalTime(val reflect.Value) cf.Value {
	time := val.Interface().(time.Time)
	return cf.Date(time)
}

func (p *Encoder) marshal(val reflect.Value) cf.Value {
	if !val.IsValid() {
		return nil
	}

	if receiver, can := implementsInterface(val, plistMarshalerType); can {
		return p.marshalPlistInterface(receiver.(Marshaler))
	}

	// time.Time implements TextMarshaler, but we need to store it in RFC3339
	if val.Type() == timeType {
		return p.marshalTime(val)
	}
	if val.Kind() == reflect.Ptr || (val.Kind() == reflect.Interface && val.NumMethod() == 0) {
		ival := val.Elem()
		if ival.IsValid() && ival.Type() == timeType {
			return p.marshalTime(ival)
		}
	}

	// Check for text marshaler.
	if receiver, can := implementsInterface(val, textMarshalerType); can {
		return p.marshalTextInterface(receiver.(encoding.TextMarshaler))
	}

	// Descend into pointers or interfaces
	if val.Kind() == reflect.Ptr || (val.Kind() == reflect.Interface && val.NumMethod() == 0) {
		val = val.Elem()
	}

	// We got this far and still may have an invalid anything or nil ptr/interface
	if !val.IsValid() || ((val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface) && val.IsNil()) {
		return nil
	}

	typ := val.Type()

	if typ == uidType {
		return cf.UID(val.Uint())
	}

	if val.Kind() == reflect.Struct {
		return p.marshalStruct(typ, val)
	}

	switch val.Kind() {
	case reflect.String:
		return cf.String(val.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &cf.Number{Signed: true, Value: uint64(val.Int())}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return &cf.Number{Signed: false, Value: val.Uint()}
	case reflect.Float32:
		return &cf.Real{Wide: false, Value: val.Float()}
	case reflect.Float64:
		return &cf.Real{Wide: true, Value: val.Float()}
	case reflect.Bool:
		return cf.Boolean(val.Bool())
	case reflect.Slice, reflect.Array:
		if typ.Elem().Kind() == reflect.Uint8 {
			bytes := []byte(nil)
			if val.CanAddr() {
				bytes = val.Bytes()
			} else {
				bytes = make([]byte, val.Len())
				reflect.Copy(reflect.ValueOf(bytes), val)
			}
			return cf.Data(bytes)
		} else {
			values := make([]cf.Value, val.Len())
			for i, length := 0, val.Len(); i < length; i++ {
				if subpval := p.marshal(val.Index(i)); subpval != nil {
					values[i] = subpval
				}
			}
			return cf.Array(values)
		}
	case reflect.Map:
		if typ.Key().Kind() != reflect.String {
			panic(&unknownTypeError{typ})
		}

		l := val.Len()
		dict := &cf.Dictionary{
			Keys:   make([]string, 0, l),
			Values: make([]cf.Value, 0, l),
		}
		for _, keyv := range val.MapKeys() {
			if subpval := p.marshal(val.MapIndex(keyv)); subpval != nil {
				dict.Keys = append(dict.Keys, keyv.String())
				dict.Values = append(dict.Values, subpval)
			}
		}
		return dict
	default:
		panic(&unknownTypeError{typ})
	}
}
