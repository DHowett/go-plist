package plist

import (
	"bytes"
	"testing"
)

type EncodingTest struct {
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

var tests = []EncodingTest{
	{
		Data: struct {
			Name string
		}{
			Name: "Dustin",
		},
		ExpectedResult: "<dict><key>Name</key><string>Dustin</string></dict>",
	},
	{
		Data:           "Hello",
		ExpectedResult: "<string>Hello</string>",
	},
	{
		Data: []int{'h', 'e', 'l', 'l', 'o'},
	},
	{
		Data: true,
	},
	{
		Data: 1.0,
	},
	{
		Data: []byte("Hello base64 data!"),
	},
	{
		Data: map[string]interface{}{
			"float":  1.0,
			"uint64": uint64(1),
		},
	},
	{
		Data:       map[int]string{1: "hi"},
		ShouldFail: true,
	},
	{
		Data: &SparseBundleHeader{
			InfoDictionaryVersion: "6.0",
			BandSize:              8388608,
			Size:                  4 * 1048576 * 1024 * 1024,
			DiskImageBundleType:   "com.apple.diskimage.sparsebundle",
			BackingStoreVersion:   1,
		},
		ExpectedResult: "<dict><key>CFBundleInfoDictionaryVersion</key><string>6.0</string><key>band-size</key><integer>8388608</integer><key>bundle-backingstore-version</key><integer>1</integer><key>diskimage-bundle-type</key><string>com.apple.diskimage.sparsebundle</string><key>size</key><integer>4398046511104</integer></dict>",
	},
	{
		Data: [][]byte{
			[]byte("Hello"),
			[]byte("World"),
		},
		ExpectedResult: "<array><data>SGVsbG8=</data><data>V29ybGQ=</data></array>",
	},
}

func TestEncode(t *testing.T) {
	for _, test := range tests {
		buf := &bytes.Buffer{}
		encoder := NewEncoder(buf)
		err := encoder.Encode(test.Data)
		t.Logf("Encoding %v", test.Data)
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

		if (test.ExpectedResult != "" && test.ExpectedResult != buf.String()) || (test.ShouldFail && err == nil) {
			t.Log("FAILED")
			t.Fail()
		} else {
			t.Log("SUCCEEDED")
		}
	}
}
