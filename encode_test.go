package plist

import (
	"bytes"
	"math"
	"testing"
	"time"
)

type EncodingTest struct {
	Name        string
	Data        interface{}
	ExpectedXML string
	ExpectedBin []byte
	ShouldFail  bool
}

type SparseBundleHeader struct {
	InfoDictionaryVersion string `plist:"CFBundleInfoDictionaryVersion"`
	BandSize              uint64 `plist:"band-size"`
	BackingStoreVersion   int    `plist:"bundle-backingstore-version"`
	DiskImageBundleType   string `plist:"diskimage-bundle-type"`
	Size                  uint64 `plist:"size"`
}

var xmlPreamble string = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">`

var tests = []EncodingTest{
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
		ExpectedXML: xmlPreamble + `<plist version="1.0"><dict><key>Name</key><string>Dustin</string></dict></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 209, 1, 2, 84, 78, 97, 109, 101, 86, 68, 117, 115, 116, 105, 110, 8, 11, 16, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 23},
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
		Name: "Map (containing nil)",
		Data: map[string]interface{}{
			"float":  1.5,
			"uint64": uint64(1),
			"nil":    nil,
		},
		ExpectedXML: xmlPreamble + `<plist version="1.0"><dict><key>float</key><real>1.5</real><key>uint64</key><integer>1</integer></dict></plist>`,
		ExpectedBin: []byte{98, 112, 108, 105, 115, 116, 48, 48, 210, 1, 3, 2, 4, 85, 102, 108, 111, 97, 116, 35, 63, 248, 0, 0, 0, 0, 0, 0, 86, 117, 105, 110, 116, 54, 52, 16, 1, 8, 13, 19, 28, 35, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 37},
	},
	{
		Name:       "Map (integer keys) (expected to fail)",
		Data:       map[int]string{1: "hi"},
		ShouldFail: true,
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
