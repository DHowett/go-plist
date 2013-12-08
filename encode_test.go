package plist

import (
	"bytes"
	"fmt"
	"testing"
)

func BenchmarkXMLEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewEncoder(&bytes.Buffer{}).Encode(plistValueTreeRawData)
	}
}

func BenchmarkBplistEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewBinaryEncoder(&bytes.Buffer{}).Encode(plistValueTreeRawData)
	}
}

func BenchmarkOpenStepEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewOpenStepEncoder(&bytes.Buffer{}).Encode(plistValueTreeRawData)
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
					t.Logf("Expected X: %s\n", test.ExpectedXML)
				}
				if test.ExpectedBin != nil {
					t.Logf("Expected B: %#v\n", test.ExpectedBin)
				}
			}

			if err == nil {
				t.Logf("Received X: %s\n", buf.String())
				t.Logf("Received B: %#v\n", bbuf.Bytes())
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
