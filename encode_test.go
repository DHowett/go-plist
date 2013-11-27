package plist

import (
	"bytes"
	"math"
	"testing"
	"time"
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
		Data: "Hello",
		ExpectedResult: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><string>Hello</string></plist>`,
	},
	{
		Data: struct {
			Name string
		}{
			Name: "Dustin",
		},
		ExpectedResult: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict><key>Name</key><string>Dustin</string></dict></plist>`,
	},
	{
		Data: []byte{'h', 'e', 'l', 'l', 'o'},
		ExpectedResult: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><data>aGVsbG8=</data></plist>`,
	},
	{
		Data: []int{'h', 'e', 'l', 'l', 'o'},
		ExpectedResult: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><array><integer>104</integer><integer>101</integer><integer>108</integer><integer>108</integer><integer>111</integer></array></plist>`,
	},
	{
		Data: true,
		ExpectedResult: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><true></true></plist>`,
	},
	{
		Data: 1.0,
		ExpectedResult: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><real>1</real></plist>`,
	},
	{
		Data: map[string]interface{}{
			"float":  1.0,
			"uint64": uint64(1),
		},
		ExpectedResult: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict><key>float</key><real>1</real><key>uint64</key><integer>1</integer></dict></plist>`,
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
		ExpectedResult: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict><key>CFBundleInfoDictionaryVersion</key><string>6.0</string><key>band-size</key><integer>8388608</integer><key>bundle-backingstore-version</key><integer>1</integer><key>diskimage-bundle-type</key><string>com.apple.diskimage.sparsebundle</string><key>size</key><integer>4398046511104</integer></dict></plist>`,
	},
	{
		Data: [][]byte{
			[]byte("Hello"),
			[]byte("World"),
		},
		ExpectedResult: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><array><data>SGVsbG8=</data><data>V29ybGQ=</data></array></plist>`,
	},
	{
		Data: time.Date(2013, 11, 27, 0, 34, 0, 0, time.UTC),
		ExpectedResult: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><date>2013-11-27T00:34:00Z</date></plist>`,
	},
	{
		Data: math.NaN(),
		ExpectedResult: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><real>nan</real></plist>`,
	},
}

func TestEncode(t *testing.T) {
	for _, test := range tests {
		buf := &bytes.Buffer{}
		encoder := NewEncoder(buf)
		err := encoder.Encode(test.Data)
		t.Logf("Encoding %#v", test.Data)

		if (test.ExpectedResult != "" && test.ExpectedResult != buf.String()) || (test.ShouldFail && err == nil) {
			if test.ShouldFail {
				t.Logf("Expected: Error")
			} else {
				if test.ExpectedResult != "" {
					t.Log("Expected:", test.ExpectedResult)
				}
			}
			t.Log("FAILED")
			t.Fail()
			if err == nil {
				t.Log("Received:", buf.String())
			} else {
				t.Log("   Error:", err)
			}
		} else {
			t.Log("SUCCEEDED")
		}
	}
}
