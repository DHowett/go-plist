package plist

import (
	"reflect"
	"testing"
	"time"
)

func BenchmarkStructMarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		e := &Encoder{}
		e.marshal(reflect.ValueOf(plistValueTreeRawData))
	}
}

func BenchmarkMapMarshal(b *testing.B) {
	data := map[string]interface{}{
		"intarray": []interface{}{
			int(1),
			int8(8),
			int16(16),
			int32(32),
			int64(64),
			uint(2),
			uint8(9),
			uint16(17),
			uint32(33),
			uint64(65),
		},
		"floats": []interface{}{
			float32(32.0),
			float64(64.0),
		},
		"booleans": []bool{
			true,
			false,
		},
		"strings": []string{
			"Hello, ASCII",
			"Hello, 世界",
		},
		"data": []byte{1, 2, 3, 4},
		"date": time.Date(2013, 11, 27, 0, 34, 0, 0, time.UTC),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e := &Encoder{}
		e.marshal(reflect.ValueOf(data))
	}
}

func TestInvalidMarshal(t *testing.T) {
	tests := []struct {
		Name  string
		Thing interface{}
	}{
		{"Function", func() {}},
		{"Nil", nil},
		{"Map with integer keys", map[int]string{1: "hi"}},
		{"Channel", make(chan int)},
	}

	for _, v := range tests {
		subtest(t, v.Name, func(t *testing.T) {
			data, err := Marshal(v.Thing, OpenStepFormat)
			if err == nil {
				t.Fatalf("expected error; got plist data: %x", data)
			} else {
				t.Log(err)
			}
		})
	}
}

type Cat struct{}

func (c *Cat) MarshalPlist() (interface{}, error) {
	return "cat", nil
}

func TestInterfaceMarshal(t *testing.T) {
	var c Cat
	b, err := Marshal(&c, XMLFormat)
	if err != nil {
		t.Log(err)
	} else if len(b) == 0 {
		t.Log("expect non-zero data")
	}
}

func TestInterfaceFieldMarshal(t *testing.T) {
	type X struct {
		C interface{} // C's type does not implement Marshaler
	}
	x := &X{
		C: &Cat{}, // C's value implements Marshaler
	}

	b, err := Marshal(x, XMLFormat)
	if err != nil {
		t.Log(err)
	} else if len(b) == 0 {
		t.Log("expect non-zero data")
	}
}

func TestMarshalInterfaceFieldPtrTime(t *testing.T) {
	type X struct {
		C interface{} // C's type is unknown
	}

	var sentinelTime = time.Date(2013, 11, 27, 0, 34, 0, 0, time.UTC)
	x := &X{
		C: &sentinelTime,
	}

	e := &Encoder{}
	rval := reflect.ValueOf(x)
	cf := e.marshal(rval)

	if dict, ok := cf.(*cfDictionary); ok {
		if _, ok := dict.values[0].(cfDate); !ok {
			t.Error("inner value is not a cfDate")
		}
	} else {
		t.Error("failed to marshal toplevel dictionary (?)")
	}
}

type Dog struct {
	Name string
}

type Animal interface{}

func TestInterfaceSliceMarshal(t *testing.T) {
	x := make([]Animal, 0)
	x = append(x, &Dog{Name: "dog"})

	b, err := Marshal(x, XMLFormat)
	if err != nil {
		t.Error(err)
	} else if len(b) == 0 {
		t.Error("expect non-zero data")
	}
}

func TestInterfaceGeneralSliceMarshal(t *testing.T) {
	x := make([]interface{}, 0) // accept any type
	x = append(x, &Dog{Name: "dog"}, "a string", 1, true)

	b, err := Marshal(x, XMLFormat)
	if err != nil {
		t.Error(err)
	} else if len(b) == 0 {
		t.Error("expect non-zero data")
	}
}

type CustomMarshaler struct {
	value interface{}
}

var _ Marshaler = (*CustomMarshaler)(nil)

func (c *CustomMarshaler) MarshalPlist() (interface{}, error) {
	// There are valid cases for testing *(nil).MarshalPlist, so don't blow up here.
	if c == nil {
		return nil, nil
	}
	return c.value, nil
}

func TestPlistMarshalerNil(t *testing.T) {
	// Direct non-nil value encodes
	subtest(t, "string", func(t *testing.T) {
		c := &CustomMarshaler{value: "hello world"}
		b, err := Marshal(c, XMLFormat)
		if err != nil {
			t.Error(err)
		}
		if len(b) == 0 {
			t.Error("expect non-nil")
		}

		t.Log(string(b))
		// <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
		// <plist version="1.0"><string>hello world</string></plist>
	})

	// Direct nil value correctly returns an error
	subtest(t, "nil", func(t *testing.T) {
		c := &CustomMarshaler{}
		b, err := Marshal(c, XMLFormat)
		if err == nil {
			t.Error("expect error")
		} else {
			t.Log(err)
		}
		if len(b) != 0 {
			t.Error("expect nil")
		}
	})

	// Field nil value with omitempty correctly omitted
	subtest(t, "ptr-omitempty", func(t *testing.T) {
		type Structure struct {
			C *CustomMarshaler `plist:"C,omitempty"`
		}
		s := &Structure{}
		b, err := Marshal(s, XMLFormat)
		if err != nil {
			t.Error(err)
		}
		if len(b) == 0 {
			t.Error("expect non-nil")
		}
		t.Log(string(b))

		// <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
		// <plist version="1.0"><dict></dict></plist>
	})

	// Non-nil field returning marshaler nil value with omitempty should be omitted
	subtest(t, "omitempty", func(t *testing.T) {
		type Structure struct {
			C CustomMarshaler `plist:"C,omitempty"`
		}
		s := &Structure{}
		b, err := Marshal(s, XMLFormat)
		if err != nil {
			t.Error(err)
		}
		if len(b) == 0 {
			t.Error("expect non-nil")
		}
		t.Log(string(b))

		// Unmarshal to prove malformed encoding
		var dst Structure
		if _, err := Unmarshal(b, &dst); err != nil {
			t.Error(err) // plist: error parsing XML property list: missing value in dictionary
		}

		// Get key without value and no error:
		// <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
		// <plist version="1.0"><dict><key>C</key></dict></plist>

		// Expect:
		// <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
		// <plist version="1.0"><dict></dict></plist>
	})

	// Field nil value without omitempty correctly emits error
	subtest(t, "ptr", func(t *testing.T) {
		type Structure struct {
			C *CustomMarshaler
		}
		s := &Structure{}
		b, err := Marshal(s, XMLFormat)
		if err == nil {
			t.Error("expect error")
		} else {
			t.Log(err)
		}
		if len(b) != 0 {
			t.Error("expect nil")
		}
	})

	// Non-nil field returning marshaler nil value without omitempty should emit an error
	subtest(t, "direct-nil-member", func(t *testing.T) {
		type Structure struct {
			C CustomMarshaler
		}
		s := &Structure{}
		b, err := Marshal(s, XMLFormat)
		if err == nil {
			t.Error("expect error")
		} else {
			t.Log(err)
		}
		if len(b) != 0 {
			t.Error("expect nil")
		}
	})
}
