package plist

import (
	"encoding"
	"fmt"
	"reflect"
	"time"
)

type IncompatibleDecodeTypeError struct {
	Type  reflect.Type
	pKind plistKind
}

func (u *IncompatibleDecodeTypeError) Error() string {
	return fmt.Sprintf("Type mismatch: tried to decode %v into variable of type %v!", plistKindNames[u.pKind], u.Type)
}

var (
	textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

func isEmptyInterface(v reflect.Value) bool {
	return v.Kind() == reflect.Interface && v.NumMethod() == 0
}

func (p *Decoder) unmarshalTextInterface(pval *plistValue, unmarshalable encoding.TextUnmarshaler) error {
	return unmarshalable.UnmarshalText([]byte(pval.value.(string)))
}

func (p *Decoder) unmarshalTime(pval *plistValue, val reflect.Value) error {
	val.Set(reflect.ValueOf(pval.value.(time.Time)))
	return nil
}

func (p *Decoder) unmarshal(pval *plistValue, val reflect.Value) (eret error) {
	defer func() {
		if r := recover(); r != nil {
			if rerr, ok := r.(error); ok {
				eret = rerr
			}
		}
	}()

	if pval == nil {
		return nil
	}

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			val.Set(reflect.New(val.Type().Elem()))
		}
		val = val.Elem()
	}

	if isEmptyInterface(val) {
		v, err := p.valueInterface(pval)
		if err != nil {
			return err
		}
		val.Set(reflect.ValueOf(v))
		return nil
	}

	incompatibleTypeError := &IncompatibleDecodeTypeError{val.Type(), pval.kind}

	// time.Time implements TextMarshaler, but we need to parse it as RFC3339
	if pval.kind == Date {
		if val.Type() == timeType {
			return p.unmarshalTime(pval, val)
		} else {
			return incompatibleTypeError
		}
	}

	if val.CanInterface() && val.Type().Implements(textUnmarshalerType) {
		return p.unmarshalTextInterface(pval, val.Interface().(encoding.TextUnmarshaler))
	}

	if val.CanAddr() {
		pv := val.Addr()
		if pv.CanInterface() && pv.Type().Implements(textUnmarshalerType) {
			return p.unmarshalTextInterface(pval, pv.Interface().(encoding.TextUnmarshaler))
		}
	}

	typ := val.Type()

	switch pval.kind {
	case String:
		if val.Kind() == reflect.String {
			val.Set(reflect.ValueOf(pval.value.(string)))
		} else {
			return incompatibleTypeError
		}
	case Integer:
		switch val.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			val.SetInt(int64(pval.value.(uint64)))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			val.SetUint(pval.value.(uint64))
		default:
			return incompatibleTypeError
		}
	case Real:
		if val.Kind() == reflect.Float32 || val.Kind() == reflect.Float64 {
			val.Set(reflect.ValueOf(pval.value.(float64)))
		} else {
			return incompatibleTypeError
		}
	case Boolean:
		if val.Kind() == reflect.Bool {
			val.Set(reflect.ValueOf(pval.value.(bool)))
		} else {
			return incompatibleTypeError
		}
	case Data:
		if typ.Elem().Kind() == reflect.Uint8 {
			val.Set(reflect.ValueOf(pval.value.([]byte)))
		} else {
			return incompatibleTypeError
		}
	case Array:
		p.unmarshalArray(pval, val)
	case Dictionary:
		p.unmarshalDictionary(pval, val)
	default:
	}

	return nil
}

func (p *Decoder) unmarshalArray(pval *plistValue, val reflect.Value) error {
	subvalues := pval.value.([]*plistValue)

	// Slice of element values.
	// Grow slice.
	cnt := len(subvalues) + val.Len()
	if cnt >= val.Cap() {
		ncap := 2 * cnt
		if ncap < 4 {
			ncap = 4
		}
		new := reflect.MakeSlice(val.Type(), val.Len(), ncap)
		reflect.Copy(new, val)
		val.Set(new)
	}
	n := val.Len()

	// Recur to read element into slice.
	for _, sval := range subvalues {
		val.SetLen(n + 1)
		if err := p.unmarshal(sval, val.Index(n)); err != nil {
			val.SetLen(n)
			return err
		}
		n++
	}
	return nil
}

func (p *Decoder) unmarshalDictionary(pval *plistValue, val reflect.Value) error {
	typ := val.Type()
	switch val.Kind() {
	case reflect.Struct:
		tinfo, err := getTypeInfo(typ)
		if err != nil {
			return err
		}

		subvalues := pval.value.(map[string]*plistValue)
		for _, finfo := range tinfo.fields {
			err := p.unmarshal(subvalues[finfo.name], finfo.value(val))
			if err != nil {
				return err
			}
		}
		return nil
	case reflect.Map:
		if val.IsNil() {
			val.Set(reflect.MakeMap(typ))
		}

		subvalues := pval.value.(map[string]*plistValue)
		for k, sval := range subvalues {
			keyv := reflect.ValueOf(k).Convert(typ.Key())
			mapElem := val.MapIndex(keyv)
			if !mapElem.IsValid() {
				mapElem = reflect.New(typ.Elem()).Elem()
			}

			err := p.unmarshal(sval, mapElem)
			if err != nil {
				return err
			}

			val.SetMapIndex(keyv, mapElem)
		}
		return nil
	default:
		return &IncompatibleDecodeTypeError{typ, pval.kind}
	}
}

/* *Interface is modelled after encoding/json */
func (p *Decoder) valueInterface(pval *plistValue) (interface{}, error) {
	switch pval.kind {
	case String:
		return pval.value.(string), nil
	case Integer:
		return pval.value.(uint64), nil
	case Real:
		return pval.value.(float64), nil
	case Boolean:
		return pval.value.(bool), nil
	case Array:
		return p.arrayInterface(pval.value.([]*plistValue))
	case Dictionary:
		return p.mapInterface(pval.value.(map[string]*plistValue))
	case Data:
		return pval.value.([]byte), nil
	case Date:
		return pval.value.(time.Time), nil
	default:
		return nil, fmt.Errorf("Unknown plist type %v", plistKindNames[pval.kind])
	}
}

func (p *Decoder) arrayInterface(subvalues []*plistValue) ([]interface{}, error) {
	out := make([]interface{}, len(subvalues))
	for i, subv := range subvalues {
		sv, err := p.valueInterface(subv)
		if err != nil {
			return nil, err
		}
		out[i] = sv
	}
	return out, nil
}

func (p *Decoder) mapInterface(subvalues map[string]*plistValue) (map[string]interface{}, error) {
	out := make(map[string]interface{})
	for k, subv := range subvalues {
		sv, err := p.valueInterface(subv)
		if err != nil {
			return nil, err
		}
		out[k] = sv
	}
	return out, nil
}
