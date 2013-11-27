package plist

import (
	"encoding"
	"reflect"
	"time"
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
	textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	timeType          = reflect.TypeOf((*time.Time)(nil)).Elem()
)

func (p *Encoder) marshalTextInterface(marshalable encoding.TextMarshaler) (*plistValue, error) {
	s, err := marshalable.MarshalText()
	if err != nil {
		return nil, err
	}
	return &plistValue{String, s}, nil
}

func (p *Encoder) marshalStruct(typ reflect.Type, val reflect.Value) (*plistValue, error) {
	tinfo, _ := getTypeInfo(typ)

	subvalues := make(map[string]*plistValue, len(tinfo.fields))
	for _, finfo := range tinfo.fields {
		value := finfo.value(val)
		if !value.IsValid() || finfo.omitEmpty && isEmptyValue(value) {
			continue
		}
		v, err := p.marshal(value)
		if err != nil {
			return nil, err
		}

		subvalues[finfo.name] = v
	}

	return &plistValue{Dictionary, subvalues}, nil
}

func (p *Encoder) marshalTime(val reflect.Value) (*plistValue, error) {
	time := val.Interface().(time.Time)
	return &plistValue{Date, time}, nil
}

func (p *Encoder) marshal(val reflect.Value) (*plistValue, error) {
	typ := val.Type()

	if !val.IsValid() {
		return nil, nil
	}

	// time.Time implements TextMarshaler, but we need to store it in RFC3339
	if val.Type() == timeType {
		return p.marshalTime(val)
	}
	if val.Kind() == reflect.Ptr || (val.Kind() == reflect.Interface && val.NumMethod() == 0) {
		ival := val.Elem()
		if ival.Type() == timeType {
			return p.marshalTime(ival)
		}
	}

	// Check for text marshaler.
	if val.CanInterface() && typ.Implements(textMarshalerType) {
		return p.marshalTextInterface(val.Interface().(encoding.TextMarshaler))
	}
	if val.CanAddr() {
		pv := val.Addr()
		if pv.CanInterface() && pv.Type().Implements(textMarshalerType) {
			return p.marshalTextInterface(pv.Interface().(encoding.TextMarshaler))
		}
	}

	// Descend into pointers or interfaces
	if val.Kind() == reflect.Ptr || (val.Kind() == reflect.Interface && val.NumMethod() == 0) {
		val = val.Elem()
		typ = val.Type()
	}

	if val.Kind() == reflect.Struct {
		return p.marshalStruct(typ, val)
	}

	switch val.Kind() {
	case reflect.String:
		return &plistValue{String, val.String()}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &plistValue{Integer, val.Int()}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return &plistValue{Integer, val.Uint()}, nil
	case reflect.Float32, reflect.Float64:
		return &plistValue{Real, val.Float()}, nil
	case reflect.Bool:
		return &plistValue{Boolean, val.Bool()}, nil
	case reflect.Slice, reflect.Array:
		if typ.Elem().Kind() == reflect.Uint8 {
			bytes := []byte(nil)
			if val.CanAddr() {
				bytes = val.Bytes()
			} else {
				bytes = make([]byte, val.Len())
				reflect.Copy(reflect.ValueOf(bytes), val)
			}
			return &plistValue{Data, bytes}, nil
		} else {
			subvalues := make([]*plistValue, val.Len())
			for idx, length := 0, val.Len(); idx < length; idx++ {
				v, err := p.marshal(val.Index(idx))
				if err != nil {
					return nil, err
				}
				subvalues[idx] = v
			}
			return &plistValue{Array, subvalues}, nil
		}
	case reflect.Map:
		if typ.Key().Kind() != reflect.String {
			return nil, &unknownTypeError{typ}
		}

		subvalues := make(map[string]*plistValue, val.Len())
		for _, keyv := range val.MapKeys() {
			v, err := p.marshal(val.MapIndex(keyv))
			if err != nil {
				return nil, err
			}

			subvalues[keyv.String()] = v
		}
		return &plistValue{Dictionary, subvalues}, nil
	default:
		return nil, &unknownTypeError{typ}
	}
	return nil, nil
}
