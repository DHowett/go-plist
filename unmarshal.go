package plist

import (
	"encoding"
	"fmt"
	"reflect"
	"runtime"
	"time"

	"howett.net/plist/cf"
)

type incompatibleDecodeTypeError struct {
	dest reflect.Type
	src  string // type name (from cf.Value)
}

func (u *incompatibleDecodeTypeError) Error() string {
	return fmt.Sprintf("plist: type mismatch: tried to decode plist type `%v' into value of type `%v'", u.src, u.dest)
}

var (
	plistUnmarshalerType = reflect.TypeOf((*Unmarshaler)(nil)).Elem()
	textUnmarshalerType  = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
	uidType              = reflect.TypeOf(UID(0))
)

func isEmptyInterface(v reflect.Value) bool {
	return v.Kind() == reflect.Interface && v.NumMethod() == 0
}

func (p *Decoder) unmarshalPlistInterface(pval cf.Value, unmarshalable Unmarshaler) {
	err := unmarshalable.UnmarshalPlist(func(i interface{}) (err error) {
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(runtime.Error); ok {
					panic(r)
				}
				err = r.(error)
			}
		}()
		p.unmarshal(pval, reflect.ValueOf(i))
		return
	})

	if err != nil {
		panic(err)
	}
}

func (p *Decoder) unmarshalTextInterface(pval cf.String, unmarshalable encoding.TextUnmarshaler) {
	err := unmarshalable.UnmarshalText([]byte(pval))
	if err != nil {
		panic(err)
	}
}

func (p *Decoder) unmarshalTime(pval cf.Date, val reflect.Value) {
	val.Set(reflect.ValueOf(time.Time(pval)))
}

func (p *Decoder) unmarshalLaxString(s string, val reflect.Value) {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i := mustParseInt(s, 10, 64)
		val.SetInt(i)
		return
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		i := mustParseUint(s, 10, 64)
		val.SetUint(i)
		return
	case reflect.Float32, reflect.Float64:
		f := mustParseFloat(s, 64)
		val.SetFloat(f)
		return
	case reflect.Bool:
		b := mustParseBool(s)
		val.SetBool(b)
		return
	case reflect.Struct:
		if val.Type() == timeType {
			t, err := time.Parse(textPlistTimeLayout, s)
			if err != nil {
				panic(err)
			}
			val.Set(reflect.ValueOf(t.In(time.UTC)))
			return
		}
		fallthrough
	default:
		panic(&incompatibleDecodeTypeError{val.Type(), "string"})
	}
}

func (p *Decoder) unmarshal(pval cf.Value, val reflect.Value) {
	if pval == nil {
		return
	}

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			val.Set(reflect.New(val.Type().Elem()))
		}
		val = val.Elem()
	}

	if isEmptyInterface(val) {
		v := p.valueInterface(pval)
		val.Set(reflect.ValueOf(v))
		return
	}

	incompatibleTypeError := &incompatibleDecodeTypeError{val.Type(), pval.TypeName()}

	// time.Time implements TextMarshaler, but we need to parse it as RFC3339
	if date, ok := pval.(cf.Date); ok {
		if val.Type() == timeType {
			p.unmarshalTime(date, val)
			return
		}
		panic(incompatibleTypeError)
	}

	if receiver, can := implementsInterface(val, plistUnmarshalerType); can {
		p.unmarshalPlistInterface(pval, receiver.(Unmarshaler))
		return
	}

	if val.Type() != timeType {
		if receiver, can := implementsInterface(val, textUnmarshalerType); can {
			if str, ok := pval.(cf.String); ok {
				p.unmarshalTextInterface(str, receiver.(encoding.TextUnmarshaler))
			} else {
				panic(incompatibleTypeError)
			}
			return
		}
	}

	typ := val.Type()

	switch pval := pval.(type) {
	case cf.String:
		if val.Kind() == reflect.String {
			val.SetString(string(pval))
			return
		}
		if p.lax {
			p.unmarshalLaxString(string(pval), val)
			return
		}

		panic(incompatibleTypeError)
	case *cf.Number:
		switch val.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			val.SetInt(int64(pval.Value))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			val.SetUint(pval.Value)
		default:
			panic(incompatibleTypeError)
		}
	case *cf.Real:
		if val.Kind() == reflect.Float32 || val.Kind() == reflect.Float64 {
			// TODO: Consider warning on a downcast (storing a 64-bit Value in a 32-bit reflect)
			val.SetFloat(pval.Value)
		} else {
			panic(incompatibleTypeError)
		}
	case cf.Boolean:
		if val.Kind() == reflect.Bool {
			val.SetBool(bool(pval))
		} else {
			panic(incompatibleTypeError)
		}
	case cf.Data:
		if val.Kind() == reflect.Slice && typ.Elem().Kind() == reflect.Uint8 {
			val.SetBytes([]byte(pval))
		} else {
			panic(incompatibleTypeError)
		}
	case cf.UID:
		if val.Type() == uidType {
			val.SetUint(uint64(pval))
		} else {
			switch val.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				val.SetInt(int64(pval))
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
				val.SetUint(uint64(pval))
			default:
				panic(incompatibleTypeError)
			}
		}
	case *cf.Array:
		p.unmarshalArray(pval, val)
	case *cf.Dictionary:
		p.unmarshalDictionary(pval, val)
	}
}

func (p *Decoder) unmarshalArray(a *cf.Array, val reflect.Value) {
	var n int
	if val.Kind() == reflect.Slice {
		// Slice of element values.
		// Grow slice.
		cnt := len(a.Values) + val.Len()
		if cnt >= val.Cap() {
			ncap := 2 * cnt
			if ncap < 4 {
				ncap = 4
			}
			new := reflect.MakeSlice(val.Type(), val.Len(), ncap)
			reflect.Copy(new, val)
			val.Set(new)
		}
		n = val.Len()
		val.SetLen(cnt)
	} else if val.Kind() == reflect.Array {
		if len(a.Values) > val.Cap() {
			panic(fmt.Errorf("plist: attempted to unmarshal %d values into an array of size %d", len(a.Values), val.Cap()))
		}
	} else {
		panic(&incompatibleDecodeTypeError{val.Type(), a.TypeName()})
	}

	// Recur to read element into slice.
	a.Range(func(i int, sval cf.Value) {
		p.unmarshal(sval, val.Index(n))
		n++
	})
	return
}

func (p *Decoder) unmarshalDictionary(dict *cf.Dictionary, val reflect.Value) {
	typ := val.Type()
	switch val.Kind() {
	case reflect.Struct:
		tinfo, err := getTypeInfo(typ)
		if err != nil {
			panic(err)
		}

		entries := make(map[string]cf.Value, dict.Len())
		dict.Range(func(i int, k string, sval cf.Value) {
			entries[k] = sval
		})

		for _, finfo := range tinfo.fields {
			p.unmarshal(entries[finfo.name], finfo.value(val))
		}
	case reflect.Map:
		if val.IsNil() {
			val.Set(reflect.MakeMap(typ))
		}

		dict.Range(func(i int, k string, sval cf.Value) {
			keyv := reflect.ValueOf(k).Convert(typ.Key())
			mapElem := val.MapIndex(keyv)
			if !mapElem.IsValid() {
				mapElem = reflect.New(typ.Elem()).Elem()
			}

			p.unmarshal(sval, mapElem)
			val.SetMapIndex(keyv, mapElem)
		})
	default:
		panic(&incompatibleDecodeTypeError{typ, dict.TypeName()})
	}
}

/* *Interface is modelled after encoding/json */
func (p *Decoder) valueInterface(pval cf.Value) interface{} {
	switch pval := pval.(type) {
	case cf.String:
		return string(pval)
	case *cf.Number:
		if pval.Signed {
			return int64(pval.Value)
		}
		return pval.Value
	case *cf.Real:
		if pval.Wide {
			return pval.Value
		} else {
			return float32(pval.Value)
		}
	case cf.Boolean:
		return bool(pval)
	case *cf.Array:
		return p.arrayInterface(pval)
	case *cf.Dictionary:
		return p.dictionaryInterface(pval)
	case cf.Data:
		return []byte(pval)
	case cf.Date:
		return time.Time(pval)
	case cf.UID:
		return UID(pval)
	}
	return nil
}

func (p *Decoder) arrayInterface(a *cf.Array) []interface{} {
	out := make([]interface{}, len(a.Values))
	a.Range(func(i int, subv cf.Value) {
		out[i] = p.valueInterface(subv)
	})
	return out
}

func (p *Decoder) dictionaryInterface(dict *cf.Dictionary) map[string]interface{} {
	out := make(map[string]interface{})
	dict.Range(func(i int, k string, subv cf.Value) {
		out[k] = p.valueInterface(subv)
	})
	return out
}
