package plist

import (
	"math"
	"reflect"
	"time"
)

type nilWriter int

func (w nilWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

type TestData struct {
	Name          string
	Data          interface{}
	DecodeData    interface{}
	ExpectedXML   string
	ExpectedBin   []byte
	ShouldFail    bool
	SkipDecode    bool
	SkipDecodeXML bool
}

type SparseBundleHeader struct {
	InfoDictionaryVersion string `plist:"CFBundleInfoDictionaryVersion"`
	BandSize              uint64 `plist:"band-size"`
	BackingStoreVersion   int    `plist:"bundle-backingstore-version"`
	DiskImageBundleType   string `plist:"diskimage-bundle-type"`
	Size                  uint64 `plist:"size"`
}

type EmbedA struct {
	EmbedC
	EmbedB EmbedB
	FieldA string
}

type EmbedB struct {
	FieldB string
	*EmbedC
}

type EmbedC struct {
	FieldA1 string `plist:"FieldA"`
	FieldA2 string
	FieldB  string
	FieldC  string
}

type TextMarshalingBool struct {
	b bool
}

func (b TextMarshalingBool) MarshalText() ([]byte, error) {
	if b.b {
		return []byte("truthful"), nil
	}
	return []byte("non-factual"), nil
}

func (b *TextMarshalingBool) UnmarshalText(text []byte) error {
	if string(text) == "truthful" {
		b.b = true
	}
	return nil
}

type TextMarshalingBoolViaPointer struct {
	b bool
}

func (b *TextMarshalingBoolViaPointer) MarshalText() ([]byte, error) {
	if b.b {
		return []byte("plausible"), nil
	}
	return []byte("unimaginable"), nil
}

func (b *TextMarshalingBoolViaPointer) UnmarshalText(text []byte) error {
	if string(text) == "plausible" {
		b.b = true
	}
	return nil
}

var xmlPreamble string = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">`

var tests = []TestData{
	{
		Name:       "Nil",
		Data:       nil,
		ShouldFail: true,
	},
	{
		Name:        "String",
		Data:        "Hello",
		ExpectedXML: xmlPreamble + `<plist version="1.0"><string>Hello</string></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 85, 72, 101, 108, 108, 111, 8, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 14},
	},
	{
		Name: "Basic Structure",
		Data: struct {
			Name string
		}{
			Name: "Dustin",
		},
		ExpectedXML: xmlPreamble + `<plist version="1.0"><dict><key>Name</key><string>Dustin</string></dict></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 209, 1, 2, 84, 78, 97, 109, 101, 86, 68, 117, 115, 116, 105, 110, 8, 11, 16, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 23},
	},
	{
		Name: "Basic Structure with non-exported fields",
		Data: struct {
			Name string
			age  int
		}{
			Name: "Dustin",
			age:  24,
		},
		DecodeData: struct {
			Name string
			age  int
		}{
			Name: "Dustin",
		},
		ExpectedXML: xmlPreamble + `<plist version="1.0"><dict><key>Name</key><string>Dustin</string></dict></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 209, 1, 2, 84, 78, 97, 109, 101, 86, 68, 117, 115, 116, 105, 110, 8, 11, 16, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 23},
	},
	{
		Name: "Basic Structure with omitted fields",
		Data: struct {
			Name string
			Age  int `plist:"-"`
		}{
			Name: "Dustin",
			Age:  24,
		},
		DecodeData: struct {
			Name string
			Age  int `plist:"-"`
		}{
			Name: "Dustin",
		},
		ExpectedXML: xmlPreamble + `<plist version="1.0"><dict><key>Name</key><string>Dustin</string></dict></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 209, 1, 2, 84, 78, 97, 109, 101, 86, 68, 117, 115, 116, 105, 110, 8, 11, 16, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 23},
	},
	{
		Name: "Basic Structure with empty omitempty fields",
		Data: struct {
			Name      string
			Age       int     `plist:"age,omitempty"`
			Slice     []int   `plist:",omitempty"`
			Bool      bool    `plist:",omitempty"`
			Uint      uint    `plist:",omitempty"`
			Float32   float32 `plist:",omitempty"`
			Float64   float64 `plist:",omitempty"`
			Stringptr *string `plist:",omitempty"`
			Notempty  uint    `plist:",omitempty"`
		}{
			Name:     "Dustin",
			Notempty: 10,
		},
		DecodeData: struct {
			Name      string
			Age       int     `plist:"age,omitempty"`
			Slice     []int   `plist:",omitempty"`
			Bool      bool    `plist:",omitempty"`
			Uint      uint    `plist:",omitempty"`
			Float32   float32 `plist:",omitempty"`
			Float64   float64 `plist:",omitempty"`
			Stringptr *string `plist:",omitempty"`
			Notempty  uint    `plist:",omitempty"`
		}{
			Name:     "Dustin",
			Notempty: 10,
		},
		ExpectedXML: xmlPreamble + `<plist version="1.0"><dict><key>Name</key><string>Dustin</string><key>Notempty</key><integer>10</integer></dict></plist>`,
		ExpectedBin: []byte{0x62, 0x70, 0x6c, 0x69, 0x73, 0x74, 0x30, 0x30, 0xd2, 0x1, 0x2, 0x3, 0x4, 0x54, 0x4e, 0x61, 0x6d, 0x65, 0x58, 0x4e, 0x6f, 0x74, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x56, 0x44, 0x75, 0x73, 0x74, 0x69, 0x6e, 0x10, 0xa, 0x8, 0xd, 0x12, 0x1b, 0x22, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x24},
	},
	{
		Name: "Structure with Anonymous Embeds",
		Data: EmbedA{
			EmbedC: EmbedC{
				FieldA1: "",
				FieldA2: "",
				FieldB:  "A.C.B",
				FieldC:  "A.C.C",
			},
			EmbedB: EmbedB{
				FieldB: "A.B.B",
				EmbedC: &EmbedC{
					FieldA1: "A.B.C.A1",
					FieldA2: "A.B.C.A2",
					FieldB:  "", // Shadowed by A.B.B
					FieldC:  "A.B.C.C",
				},
			},
			FieldA: "A.A",
		},
		SkipDecode:  false,
		ExpectedXML: xmlPreamble + `<plist version="1.0"><dict><key>EmbedB</key><dict><key>FieldA</key><string>A.B.C.A1</string><key>FieldA2</key><string>A.B.C.A2</string><key>FieldB</key><string>A.B.B</string><key>FieldC</key><string>A.B.C.C</string></dict><key>FieldA</key><string>A.A</string><key>FieldA2</key><string></string><key>FieldB</key><string>A.C.B</string><key>FieldC</key><string>A.C.C</string></dict></plist>`,
		ExpectedBin: []byte{0x62, 0x70, 0x6c, 0x69, 0x73, 0x74, 0x30, 0x30, 0xd5, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0xb, 0xc, 0xd, 0xe, 0x56, 0x45, 0x6d, 0x62, 0x65, 0x64, 0x42, 0x56, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x41, 0x57, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x41, 0x32, 0x56, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x42, 0x56, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x43, 0xd4, 0x2, 0x3, 0x4, 0x5, 0x7, 0x8, 0x9, 0xa, 0x58, 0x41, 0x2e, 0x42, 0x2e, 0x43, 0x2e, 0x41, 0x31, 0x58, 0x41, 0x2e, 0x42, 0x2e, 0x43, 0x2e, 0x41, 0x32, 0x55, 0x41, 0x2e, 0x42, 0x2e, 0x42, 0x57, 0x41, 0x2e, 0x42, 0x2e, 0x43, 0x2e, 0x43, 0x53, 0x41, 0x2e, 0x41, 0x50, 0x55, 0x41, 0x2e, 0x43, 0x2e, 0x42, 0x55, 0x41, 0x2e, 0x43, 0x2e, 0x43, 0x8, 0x13, 0x1a, 0x21, 0x29, 0x30, 0x37, 0x40, 0x49, 0x52, 0x58, 0x60, 0x64, 0x65, 0x6b, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xf, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x71},
	},
	{
		Name:        "Arbitrary Byte Data",
		Data:        []byte{'h', 'e', 'l', 'l', 'o'},
		ExpectedXML: xmlPreamble + `<plist version="1.0"><data>aGVsbG8=</data></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 69, 104, 101, 108, 108, 111, 8, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 14},
	},
	{
		Name:        "Arbitrary Integer Slice",
		Data:        []int{'h', 'e', 'l', 'l', 'o'},
		ExpectedXML: xmlPreamble + `<plist version="1.0"><array><integer>104</integer><integer>101</integer><integer>108</integer><integer>108</integer><integer>111</integer></array></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 165, 1, 2, 3, 3, 4, 16, 104, 16, 101, 16, 108, 16, 111, 8, 14, 16, 18, 20, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 22},
	},
	{
		Name:        "Arbitrary Integer Array",
		Data:        [3]int{'h', 'i', '!'},
		ExpectedXML: xmlPreamble + `<plist version="1.0"><array><integer>104</integer><integer>105</integer><integer>33</integer></array></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 163, 1, 2, 3, 16, 104, 16, 105, 16, 33, 8, 12, 14, 16, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 18},
	},
	{
		Name:        "Unsigned Integers of Increasing Size",
		Data:        []uint64{0xff, 0xfff, 0xffff, 0xfffff, 0xffffff, 0xfffffff, 0xffffffff, 0xffffffffffffffff},
		ExpectedXML: xmlPreamble + `<plist version="1.0"><array><integer>255</integer><integer>4095</integer><integer>65535</integer><integer>1048575</integer><integer>16777215</integer><integer>268435455</integer><integer>4294967295</integer><integer>18446744073709551615</integer></array></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 168, 1, 2, 3, 4, 5, 6, 7, 8, 16, 255, 17, 15, 255, 17, 255, 255, 18, 0, 15, 255, 255, 18, 0, 255, 255, 255, 18, 15, 255, 255, 255, 18, 255, 255, 255, 255, 19, 255, 255, 255, 255, 255, 255, 255, 255, 8, 17, 19, 22, 25, 30, 35, 40, 45, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 9, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 54},
	},
	{
		Name:          "Floats of Increasing Bitness",
		Data:          []interface{}{float32(math.MaxFloat32), float64(math.MaxFloat64)},
		ExpectedXML:   xmlPreamble + `<plist version="1.0"><array><real>3.4028234663852886e+38</real><real>1.7976931348623157e+308</real></array></plist>`,
		ExpectedBin:   []byte{98, 112, 108, 105, 115, 116, 48, 48, 162, 1, 2, 34, 127, 127, 255, 255, 35, 127, 239, 255, 255, 255, 255, 255, 255, 8, 11, 16, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 25},
		SkipDecodeXML: true,
	},
	{
		Name:        "Boolean True",
		Data:        true,
		ExpectedXML: xmlPreamble + `<plist version="1.0"><true></true></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 9, 8, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 9},
	},
	{
		Name:        "Floating-Point Value",
		Data:        3.14159265358979323846264338327950288,
		ExpectedXML: xmlPreamble + `<plist version="1.0"><real>3.141592653589793</real></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 35, 64, 9, 33, 251, 84, 68, 45, 24, 8, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 17},
	},
	{
		Name: "Map (containing arbitrary types)",
		Data: map[string]interface{}{
			"float":  1.0,
			"uint64": uint64(1),
		},
		ExpectedXML: xmlPreamble + `<plist version="1.0"><dict><key>float</key><real>1</real><key>uint64</key><integer>1</integer></dict></plist>`,
		ExpectedBin: []byte{0x62, 0x70, 0x6c, 0x69, 0x73, 0x74, 0x30, 0x30, 0xd2, 0x1, 0x2, 0x3, 0x4, 0x55, 0x66, 0x6c, 0x6f, 0x61, 0x74, 0x56, 0x75, 0x69, 0x6e, 0x74, 0x36, 0x34, 0x23, 0x3f, 0xf0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x10, 0x1, 0x8, 0xd, 0x13, 0x1a, 0x23, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x25},
	},
	{
		Name: "Map (containing all variations of all types)",
		Data: interface{}(map[string]interface{}{
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
		}),
		ExpectedXML: xmlPreamble + `<plist version="1.0"><dict><key>booleans</key><array><true></true><false></false></array><key>data</key><data>AQIDBA==</data><key>date</key><date>2013-11-27T00:34:00Z</date><key>floats</key><array><real>32</real><real>64</real></array><key>intarray</key><array><integer>1</integer><integer>8</integer><integer>16</integer><integer>32</integer><integer>64</integer><integer>2</integer><integer>9</integer><integer>17</integer><integer>33</integer><integer>65</integer></array><key>strings</key><array><string>Hello, ASCII</string><string>Hello, 世界</string></array></dict></plist>`,
		ExpectedBin: []byte{0x62, 0x70, 0x6c, 0x69, 0x73, 0x74, 0x30, 0x30, 0xd6, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0xa, 0xb, 0xc, 0xf, 0x1a, 0x58, 0x62, 0x6f, 0x6f, 0x6c, 0x65, 0x61, 0x6e, 0x73, 0x54, 0x64, 0x61, 0x74, 0x61, 0x54, 0x64, 0x61, 0x74, 0x65, 0x56, 0x66, 0x6c, 0x6f, 0x61, 0x74, 0x73, 0x58, 0x69, 0x6e, 0x74, 0x61, 0x72, 0x72, 0x61, 0x79, 0x57, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x73, 0xa2, 0x8, 0x9, 0x9, 0x8, 0x44, 0x1, 0x2, 0x3, 0x4, 0x33, 0x41, 0xb8, 0x45, 0x75, 0x78, 0x0, 0x0, 0x0, 0xa2, 0xd, 0xe, 0x22, 0x42, 0x0, 0x0, 0x0, 0x23, 0x40, 0x50, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xaa, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x10, 0x1, 0x10, 0x8, 0x10, 0x10, 0x10, 0x20, 0x10, 0x40, 0x10, 0x2, 0x10, 0x9, 0x10, 0x11, 0x10, 0x21, 0x10, 0x41, 0xa2, 0x1b, 0x1c, 0x5c, 0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x2c, 0x20, 0x41, 0x53, 0x43, 0x49, 0x49, 0x69, 0x0, 0x48, 0x0, 0x65, 0x0, 0x6c, 0x0, 0x6c, 0x0, 0x6f, 0x0, 0x2c, 0x0, 0x20, 0x4e, 0x16, 0x75, 0x4c, 0x8, 0x15, 0x1e, 0x23, 0x28, 0x2f, 0x38, 0x40, 0x43, 0x44, 0x45, 0x4a, 0x53, 0x56, 0x5b, 0x64, 0x6f, 0x71, 0x73, 0x75, 0x77, 0x79, 0x7b, 0x7d, 0x7f, 0x81, 0x83, 0x86, 0x93, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1d, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xa6},
		SkipDecode:  true,
	},
	{
		Name: "Map (containing nil)",
		Data: map[string]interface{}{
			"float":  1.5,
			"uint64": uint64(1),
			"nil":    nil,
		},
		DecodeData: map[string]interface{}{
			"float":  1.5,
			"uint64": uint64(1),
		},
		ExpectedXML: xmlPreamble + `<plist version="1.0"><dict><key>float</key><real>1.5</real><key>uint64</key><integer>1</integer></dict></plist>`,
		ExpectedBin: []byte{0x62, 0x70, 0x6c, 0x69, 0x73, 0x74, 0x30, 0x30, 0xd2, 0x1, 0x2, 0x3, 0x4, 0x55, 0x66, 0x6c, 0x6f, 0x61, 0x74, 0x56, 0x75, 0x69, 0x6e, 0x74, 0x36, 0x34, 0x23, 0x3f, 0xf8, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x10, 0x1, 0x8, 0xd, 0x13, 0x1a, 0x23, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x25},
	},
	{
		Name:       "Map (integer keys) (expected to fail)",
		Data:       map[int]string{1: "hi"},
		ShouldFail: true,
		SkipDecode: true,
	},
	{
		Name: "Pointer to structure with plist tags",
		Data: &SparseBundleHeader{
			InfoDictionaryVersion: "6.0",
			BandSize:              8388608,
			Size:                  4 * 1048576 * 1024 * 1024,
			DiskImageBundleType:   "com.apple.diskimage.sparsebundle",
			BackingStoreVersion:   1,
		},
		ExpectedXML: xmlPreamble + `<plist version="1.0"><dict><key>CFBundleInfoDictionaryVersion</key><string>6.0</string><key>band-size</key><integer>8388608</integer><key>bundle-backingstore-version</key><integer>1</integer><key>diskimage-bundle-type</key><string>com.apple.diskimage.sparsebundle</string><key>size</key><integer>4398046511104</integer></dict></plist>`,
		ExpectedBin: []byte{0x62, 0x70, 0x6c, 0x69, 0x73, 0x74, 0x30, 0x30, 0xd5, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xa, 0x5f, 0x10, 0x1d, 0x43, 0x46, 0x42, 0x75, 0x6e, 0x64, 0x6c, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x44, 0x69, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x72, 0x79, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x59, 0x62, 0x61, 0x6e, 0x64, 0x2d, 0x73, 0x69, 0x7a, 0x65, 0x5f, 0x10, 0x1b, 0x62, 0x75, 0x6e, 0x64, 0x6c, 0x65, 0x2d, 0x62, 0x61, 0x63, 0x6b, 0x69, 0x6e, 0x67, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x2d, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x5f, 0x10, 0x15, 0x64, 0x69, 0x73, 0x6b, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x2d, 0x62, 0x75, 0x6e, 0x64, 0x6c, 0x65, 0x2d, 0x74, 0x79, 0x70, 0x65, 0x54, 0x73, 0x69, 0x7a, 0x65, 0x53, 0x36, 0x2e, 0x30, 0x12, 0x0, 0x80, 0x0, 0x0, 0x10, 0x1, 0x5f, 0x10, 0x20, 0x63, 0x6f, 0x6d, 0x2e, 0x61, 0x70, 0x70, 0x6c, 0x65, 0x2e, 0x64, 0x69, 0x73, 0x6b, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x2e, 0x73, 0x70, 0x61, 0x72, 0x73, 0x65, 0x62, 0x75, 0x6e, 0x64, 0x6c, 0x65, 0x13, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0, 0x0, 0x0, 0x8, 0x13, 0x33, 0x3d, 0x5b, 0x73, 0x78, 0x7c, 0x81, 0x83, 0xa6, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xb, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xaf},
		SkipDecode:  true,
	},
	{
		Name: "Array of byte arrays",
		Data: [][]byte{
			[]byte("Hello"),
			[]byte("World"),
		},
		ExpectedXML: xmlPreamble + `<plist version="1.0"><array><data>SGVsbG8=</data><data>V29ybGQ=</data></array></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 162, 1, 2, 69, 72, 101, 108, 108, 111, 69, 87, 111, 114, 108, 100, 8, 11, 17, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 23},
	},
	{
		Name:        "Date",
		Data:        time.Date(2013, 11, 27, 0, 34, 0, 0, time.UTC),
		ExpectedXML: xmlPreamble + `<plist version="1.0"><date>2013-11-27T00:34:00Z</date></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 51, 65, 184, 69, 117, 120, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 17},
	},
	{
		Name:        "Floating-Point NaN",
		Data:        math.NaN(),
		ExpectedXML: xmlPreamble + `<plist version="1.0"><real>nan</real></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 35, 127, 248, 0, 0, 0, 0, 0, 1, 8, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 17},
		SkipDecode:  true,
	},
	{
		Name:        "Floating-Point Infinity",
		Data:        math.Inf(1),
		ExpectedXML: xmlPreamble + `<plist version="1.0"><real>inf</real></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 35, 127, 240, 0, 0, 0, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 17},
	},
	{
		Name:        "UTF-8 string",
		Data:        []string{"Hello, ASCII", "Hello, 世界"},
		ExpectedXML: xmlPreamble + `<plist version="1.0"><array><string>Hello, ASCII</string><string>Hello, 世界</string></array></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 162, 1, 2, 92, 72, 101, 108, 108, 111, 44, 32, 65, 83, 67, 73, 73, 105, 0, 72, 0, 101, 0, 108, 0, 108, 0, 111, 0, 44, 0, 32, 78, 22, 117, 76, 8, 11, 24, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 43},
	},
	{
		Name:        "An array containing more than fifteen items",
		Data:        []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		ExpectedXML: xmlPreamble + `<plist version="1.0"><array><integer>1</integer><integer>2</integer><integer>3</integer><integer>4</integer><integer>5</integer><integer>6</integer><integer>7</integer><integer>8</integer><integer>9</integer><integer>10</integer><integer>11</integer><integer>12</integer><integer>13</integer><integer>14</integer><integer>15</integer><integer>16</integer></array></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 175, 16, 16, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 16, 1, 16, 2, 16, 3, 16, 4, 16, 5, 16, 6, 16, 7, 16, 8, 16, 9, 16, 10, 16, 11, 16, 12, 16, 13, 16, 14, 16, 15, 16, 16, 8, 27, 29, 31, 33, 35, 37, 39, 41, 43, 45, 47, 49, 51, 53, 55, 57, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 17, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 59},
	},
	{
		Name:        "TextMarshaler/TextUnmarshaler",
		Data:        TextMarshalingBool{true},
		ExpectedXML: xmlPreamble + `<plist version="1.0"><string>truthful</string></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 88, 116, 114, 117, 116, 104, 102, 117, 108, 8, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 17},
		// We expect false here because the non-pointer version cannot mutate itself.
	},
	{
		Name:        "TextMarshaler/TextUnmarshaler via Pointer",
		Data:        &TextMarshalingBoolViaPointer{false},
		ExpectedXML: xmlPreamble + `<plist version="1.0"><string>unimaginable</string></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 92, 117, 110, 105, 109, 97, 103, 105, 110, 97, 98, 108, 101, 8, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 21},
		DecodeData:  TextMarshalingBoolViaPointer{false},
	},
	{
		Name:        "Signed Integers",
		Data:        []int64{-1, -127, -255, -32767, -65535},
		ExpectedXML: xmlPreamble + `<plist version="1.0"><array><integer>-1</integer><integer>-127</integer><integer>-255</integer><integer>-32767</integer><integer>-65535</integer></array></plist>`,
		ExpectedBin: []byte{0x62, 0x70, 0x6c, 0x69, 0x73, 0x74, 0x30, 0x30, 0xa5, 0x1, 0x2, 0x3, 0x4, 0x5, 0x13, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x13, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x81, 0x13, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x1, 0x13, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x80, 0x1, 0x13, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x0, 0x1, 0x8, 0xe, 0x17, 0x20, 0x29, 0x32, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x6, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3b},
	},
}

type EverythingTestData struct {
	Intarray []uint64  `plist:"intarray"`
	Floats   []float64 `plist:"floats"`
	Booleans []bool    `plist:"booleans"`
	Strings  []string  `plist:"strings"`
	Dat      []byte    `plist:"data"`
	Date     time.Time `plist:"date"`
}

var plistValueTreeRawData *EverythingTestData = &EverythingTestData{
	Intarray: []uint64{1, 8, 16, 32, 64, 2, 9, 17, 33, 65},
	Floats:   []float64{32.0, 64.0},
	Booleans: []bool{true, false},
	Strings:  []string{"Hello, ASCII", "Hello, 世界"},
	Dat:      []byte{1, 2, 3, 4},
	Date:     time.Date(2013, 11, 27, 0, 34, 0, 0, time.UTC),
}
var plistValueTree *plistValue
var plistValueTreeAsBplist []byte = []byte{98, 112, 108, 105, 115, 116, 48, 48, 214, 1, 13, 17, 21, 25, 27, 2, 14, 18, 22, 26, 28, 88, 105, 110, 116, 97, 114, 114, 97, 121, 170, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 16, 1, 16, 8, 16, 16, 16, 32, 16, 64, 16, 2, 16, 9, 16, 17, 16, 33, 16, 65, 86, 102, 108, 111, 97, 116, 115, 162, 15, 16, 34, 66, 0, 0, 0, 35, 64, 80, 0, 0, 0, 0, 0, 0, 88, 98, 111, 111, 108, 101, 97, 110, 115, 162, 19, 20, 9, 8, 87, 115, 116, 114, 105, 110, 103, 115, 162, 23, 24, 92, 72, 101, 108, 108, 111, 44, 32, 65, 83, 67, 73, 73, 105, 0, 72, 0, 101, 0, 108, 0, 108, 0, 111, 0, 44, 0, 32, 78, 22, 117, 76, 84, 100, 97, 116, 97, 68, 1, 2, 3, 4, 84, 100, 97, 116, 101, 51, 65, 184, 69, 117, 120, 0, 0, 0, 8, 21, 30, 41, 43, 45, 47, 49, 51, 53, 55, 57, 59, 61, 68, 71, 76, 85, 94, 97, 98, 99, 107, 110, 123, 142, 147, 152, 157, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 29, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 166}
var plistValueTreeAsXML string = xmlPreamble + `<plist version="1.0"><dict><key>intarray</key><array><integer>1</integer><integer>8</integer><integer>16</integer><integer>32</integer><integer>64</integer><integer>2</integer><integer>9</integer><integer>17</integer><integer>33</integer><integer>65</integer></array><key>floats</key><array><real>32</real><real>64</real></array><key>booleans</key><array><true></true><false></false></array><key>strings</key><array><string>Hello, ASCII</string><string>Hello, 世界</string></array><key>data</key><data>AQIDBA==</data><key>date</key><date>2013-11-27T00:34:00Z</date></dict></plist>`

type LaxTestData struct {
	I64 int64
	U64 uint64
	F64 float64
	B   bool
	D   time.Time
}

var laxTestData = LaxTestData{1, 2, 3.0, true, time.Date(2013, 11, 27, 0, 34, 0, 0, time.UTC)}

func setupPlistValues() {
	plistValueTree = &plistValue{
		Dictionary,
		&dictionary{m: map[string]*plistValue{
			"intarray": &plistValue{Array, []*plistValue{
				&plistValue{Integer, signedInt{uint64(1), false}},
				&plistValue{Integer, signedInt{uint64(8), false}},
				&plistValue{Integer, signedInt{uint64(16), false}},
				&plistValue{Integer, signedInt{uint64(32), false}},
				&plistValue{Integer, signedInt{uint64(64), false}},
				&plistValue{Integer, signedInt{uint64(2), false}},
				&plistValue{Integer, signedInt{uint64(8), false}},
				&plistValue{Integer, signedInt{uint64(17), false}},
				&plistValue{Integer, signedInt{uint64(33), false}},
				&plistValue{Integer, signedInt{uint64(65), false}},
			}},
			"floats": &plistValue{Array, []*plistValue{
				&plistValue{Real, sizedFloat{float64(32.0), 32}},
				&plistValue{Real, sizedFloat{float64(64.0), 64}},
			}},
			"booleans": &plistValue{Array, []*plistValue{
				&plistValue{Boolean, true},
				&plistValue{Boolean, false},
			}},
			"strings": &plistValue{Array, []*plistValue{
				&plistValue{String, "Hello, ASCII"},
				&plistValue{String, "Hello, 世界"},
			}},
			"data": &plistValue{Data, []byte{1, 2, 3, 4}},
			"date": &plistValue{Date, time.Date(2013, 11, 27, 0, 34, 0, 0, time.UTC)},
		}},
	}
}

func init() {
	setupPlistValues()

	// Pre-warm the type info struct to remove it from benchmarking
	getTypeInfo(reflect.ValueOf(plistValueTreeRawData).Type())
}
