package plist

import (
	"bytes"
	"fmt"
	"math"
	"testing"
	"time"
)

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

type TextMarshalingBool struct {
	b bool
}

func (b TextMarshalingBool) MarshalText() ([]byte, error) {
	if b.b {
		return []byte("truthful"), nil
	}
	return []byte("non-factual"), nil
}

func (b TextMarshalingBool) UnmarshalText(text []byte) error {
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
		ExpectedBin: nil,
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
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 210, 1, 3, 2, 4, 85, 102, 108, 111, 97, 116, 35, 63, 240, 0, 0, 0, 0, 0, 0, 86, 117, 105, 110, 116, 54, 52, 16, 1, 8, 13, 19, 28, 35, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 37},
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
		ExpectedXML: xmlPreamble + `<plist version="1.0"><dict><key>intarray</key><array><integer>1</integer><integer>8</integer><integer>16</integer><integer>32</integer><integer>64</integer><integer>2</integer><integer>9</integer><integer>17</integer><integer>33</integer><integer>65</integer></array><key>floats</key><array><real>32</real><real>64</real></array><key>booleans</key><array><true></true><false></false></array><key>strings</key><array><string>Hello, ASCII</string><string>Hello, 世界</string></array><key>data</key><data>AQIDBA==</data><key>date</key><date>2013-11-27T00:34:00Z</date></dict></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 214, 1, 13, 17, 21, 25, 27, 2, 14, 18, 22, 26, 28, 88, 105, 110, 116, 97, 114, 114, 97, 121, 170, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 16, 1, 16, 8, 16, 16, 16, 32, 16, 64, 16, 2, 16, 9, 16, 17, 16, 33, 16, 65, 86, 102, 108, 111, 97, 116, 115, 162, 15, 16, 34, 66, 0, 0, 0, 35, 64, 80, 0, 0, 0, 0, 0, 0, 88, 98, 111, 111, 108, 101, 97, 110, 115, 162, 19, 20, 9, 8, 87, 115, 116, 114, 105, 110, 103, 115, 162, 23, 24, 92, 72, 101, 108, 108, 111, 44, 32, 65, 83, 67, 73, 73, 105, 0, 72, 0, 101, 0, 108, 0, 108, 0, 111, 0, 44, 0, 32, 78, 22, 117, 76, 84, 100, 97, 116, 97, 68, 1, 2, 3, 4, 84, 100, 97, 116, 101, 51, 65, 184, 69, 117, 120, 0, 0, 0, 8, 21, 30, 41, 43, 45, 47, 49, 51, 53, 55, 57, 59, 61, 68, 71, 76, 85, 94, 97, 98, 99, 107, 110, 123, 142, 147, 152, 157, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 29, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 166},
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
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 210, 1, 3, 2, 4, 85, 102, 108, 111, 97, 116, 35, 63, 248, 0, 0, 0, 0, 0, 0, 86, 117, 105, 110, 116, 54, 52, 16, 1, 8, 13, 19, 28, 35, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 37},
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
		DecodeData: TextMarshalingBool{false},
	},
	{
		Name:        "TextMarshaler/TextUnmarshaler via Pointer",
		Data:        &TextMarshalingBoolViaPointer{false},
		ExpectedXML: xmlPreamble + `<plist version="1.0"><string>unimaginable</string></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 92, 117, 110, 105, 109, 97, 103, 105, 110, 97, 98, 108, 101, 8, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 21},
		DecodeData:  TextMarshalingBoolViaPointer{false},
	},
}

func BenchmarkBinaryEncode(b *testing.B) {
	for i := 0; i < b.N/len(tests); i++ {
		for _, test := range tests {
			NewBinaryEncoder(&bytes.Buffer{}).Encode(test.Data)
		}
	}
}

func BenchmarkXMLEncode(b *testing.B) {
	for i := 0; i < b.N/len(tests); i++ {
		for _, test := range tests {
			NewEncoder(&bytes.Buffer{}).Encode(test.Data)
		}
	}
}

func TestEncode(t *testing.T) {
	var failed bool
	for _, test := range tests {
		failed = false
		buf := &bytes.Buffer{}
		encoder := NewEncoder(buf)
		err := encoder.Encode(test.Data)

		bbuf := &bytes.Buffer{}
		bencoder := NewBinaryEncoder(bbuf)
		bencoder.Encode(test.Data)

		t.Logf("Testing Encode (%s)", test.Name)

		if test.ShouldFail && err == nil {
			failed = true
		}

		if test.ExpectedXML != "" && test.ExpectedXML != buf.String() {
			failed = true
		}

		if test.ExpectedBin != nil && !bytes.Equal(test.ExpectedBin, bbuf.Bytes()) {
			failed = true
		}

		if failed {
			t.Logf("Value: %#v", test.Data)
			if test.ShouldFail {
				t.Logf("Expected: Error")
			} else {
				if test.ExpectedXML != "" {
					t.Log("Expected X:", test.ExpectedXML)
				}
				if test.ExpectedBin != nil {
					t.Log("Expected B:", test.ExpectedBin)
				}
			}

			if err == nil {
				t.Log("Received X:", buf.String())
				t.Log("Received B:", bbuf.Bytes())
			} else {
				t.Log("   Error:", err)
			}
			t.Log("FAILED")
			t.Fail()
		}
	}
}

func ExampleEncoder_Encode() {
	type sparseBundleHeader struct {
		InfoDictionaryVersion string `plist:"CFBundleInfoDictionaryVersion"`
		BandSize              uint64 `plist:"band-size"`
		BackingStoreVersion   int    `plist:"bundle-backingstore-version"`
		DiskImageBundleType   string `plist:"diskimage-bundle-type"`
		Size                  uint64 `plist:"size"`
	}
	data := &sparseBundleHeader{
		InfoDictionaryVersion: "6.0",
		BandSize:              8388608,
		Size:                  4 * 1048576 * 1024 * 1024,
		DiskImageBundleType:   "com.apple.diskimage.sparsebundle",
		BackingStoreVersion:   1,
	}

	buf := &bytes.Buffer{}
	encoder := NewEncoder(buf)
	err := encoder.Encode(data)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(buf.String())

	// Output: <?xml version="1.0" encoding="UTF-8"?>
	// <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict><key>CFBundleInfoDictionaryVersion</key><string>6.0</string><key>band-size</key><integer>8388608</integer><key>bundle-backingstore-version</key><integer>1</integer><key>diskimage-bundle-type</key><string>com.apple.diskimage.sparsebundle</string><key>size</key><integer>4398046511104</integer></dict></plist>
}
