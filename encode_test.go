package plist

import (
	"bytes"
	"math"
	"testing"
	"time"
)

type EncodingTest struct {
	Name           string
	Data           interface{}
	ExpectedResult string
	ShouldFail     bool
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
		Name:           "String",
		Data:           "Hello",
		ExpectedResult: xmlPreamble + `<plist version="1.0"><string>Hello</string></plist>`,
	},
	{
		Name: "Basic Structure",
		Data: struct {
			Name string
		}{
			Name: "Dustin",
		},
		ExpectedResult: xmlPreamble + `<plist version="1.0"><dict><key>Name</key><string>Dustin</string></dict></plist>`,
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
		ExpectedResult: xmlPreamble + `<plist version="1.0"><dict><key>Name</key><string>Dustin</string></dict></plist>`,
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
		ExpectedResult: xmlPreamble + `<plist version="1.0"><dict><key>Name</key><string>Dustin</string></dict></plist>`,
	},
	{
		Name:           "Arbitrary Byte Data",
		Data:           []byte{'h', 'e', 'l', 'l', 'o'},
		ExpectedResult: xmlPreamble + `<plist version="1.0"><data>aGVsbG8=</data></plist>`,
	},
	{
		Name:           "Arbitrary Integer Slice",
		Data:           []int{'h', 'e', 'l', 'l', 'o'},
		ExpectedResult: xmlPreamble + `<plist version="1.0"><array><integer>104</integer><integer>101</integer><integer>108</integer><integer>108</integer><integer>111</integer></array></plist>`,
	},
	{
		Name:           "Arbitrary Integer Array",
		Data:           [3]int{'h', 'i', '!'},
		ExpectedResult: xmlPreamble + `<plist version="1.0"><array><integer>104</integer><integer>105</integer><integer>33</integer></array></plist>`,
	},
	{
		Name:           "Boolean True",
		Data:           true,
		ExpectedResult: xmlPreamble + `<plist version="1.0"><true></true></plist>`,
	},
	{
		Name:           "Floating-Point Value",
		Data:           math.Pi,
		ExpectedResult: xmlPreamble + `<plist version="1.0"><real>3.141592653589793</real></plist>`,
	},
	{
		Name: "Map (containing arbitrary types)",
		Data: map[string]interface{}{
			"float":  1.0,
			"uint64": uint64(1),
		},
		ExpectedResult: xmlPreamble + `<plist version="1.0"><dict><key>float</key><real>1</real><key>uint64</key><integer>1</integer></dict></plist>`,
	},
	{
		Name: "Map (containing nil)",
		Data: map[string]interface{}{
			"float":  1.5,
			"uint64": uint64(1),
			"nil":    nil,
		},
		ExpectedResult: xmlPreamble + `<plist version="1.0"><dict><key>float</key><real>1.5</real><key>uint64</key><integer>1</integer></dict></plist>`,
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
		ExpectedResult: xmlPreamble + `<plist version="1.0"><dict><key>CFBundleInfoDictionaryVersion</key><string>6.0</string><key>band-size</key><integer>8388608</integer><key>bundle-backingstore-version</key><integer>1</integer><key>diskimage-bundle-type</key><string>com.apple.diskimage.sparsebundle</string><key>size</key><integer>4398046511104</integer></dict></plist>`,
	},
	{
		Name: "Array of byte arrays",
		Data: [][]byte{
			[]byte("Hello"),
			[]byte("World"),
		},
		ExpectedResult: xmlPreamble + `<plist version="1.0"><array><data>SGVsbG8=</data><data>V29ybGQ=</data></array></plist>`,
	},
	{
		Name:           "Date",
		Data:           time.Date(2013, 11, 27, 0, 34, 0, 0, time.UTC),
		ExpectedResult: xmlPreamble + `<plist version="1.0"><date>2013-11-27T00:34:00Z</date></plist>`,
	},
	{
		Name:           "Floating-Point NaN",
		Data:           math.NaN(),
		ExpectedResult: xmlPreamble + `<plist version="1.0"><real>nan</real></plist>`,
	},
	{
		Name:           "Floating-Point Infinity",
		Data:           math.Inf(1),
		ExpectedResult: xmlPreamble + `<plist version="1.0"><real>inf</real></plist>`,
	},
}

func TestEncode(t *testing.T) {
	for _, test := range tests {
		buf := &bytes.Buffer{}
		encoder := NewEncoder(buf)
		err := encoder.Encode(test.Data)

		t.Logf("Testing Encode (%s)", test.Name)

		if (test.ExpectedResult != "" && test.ExpectedResult != buf.String()) || (test.ShouldFail && err == nil) {
			t.Logf("Value: %#v", test.Data)
			if test.ShouldFail {
				t.Logf("Expected: Error")
			} else {
				if test.ExpectedResult != "" {
					t.Log("Expected:", test.ExpectedResult)
				}
			}

			if err == nil {
				t.Log("Received:", buf.String())
			} else {
				t.Log("   Error:", err)
			}

			t.Log("FAILED")
			t.Fail()
		}
	}
}
