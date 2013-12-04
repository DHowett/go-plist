package plist

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

func BenchmarkXMLDecode(b *testing.B) {
	ntests := 0
	for _, test := range tests {
		if test.SkipDecodeXML || test.SkipDecode || test.ExpectedXML == "" {
			continue
		}

		testData := reflect.ValueOf(test.Data)
		if !testData.IsValid() || isEmptyInterface(testData) {
			continue
		}
		ntests++
	}

	b.ResetTimer()
	for i := 0; i < b.N/ntests; i++ {
		for _, test := range tests {
			b.StopTimer()
			if test.SkipDecode || test.ExpectedXML == "" {
				continue
			}

			testData := reflect.ValueOf(test.Data)
			if !testData.IsValid() || isEmptyInterface(testData) {
				continue
			}

			var bval interface{} = reflect.New(testData.Type()).Interface()

			buf := bytes.NewReader([]byte(test.ExpectedXML))
			b.StartTimer()
			decoder := NewDecoder(buf)
			decoder.Decode(bval)
			b.StopTimer()
		}
	}
}

func BenchmarkBinaryDecode(b *testing.B) {
	ntests := 0
	for _, test := range tests {
		if test.SkipDecode || test.ExpectedBin == nil {
			continue
		}

		testData := reflect.ValueOf(test.Data)
		if !testData.IsValid() || isEmptyInterface(testData) {
			continue
		}
		ntests++
	}

	b.ResetTimer()
	for i := 0; i < b.N/ntests; i++ {
		for _, test := range tests {
			b.StopTimer()
			if test.SkipDecode || test.ExpectedBin == nil {
				continue
			}

			testData := reflect.ValueOf(test.Data)
			if !testData.IsValid() || isEmptyInterface(testData) {
				continue
			}

			var bval interface{} = reflect.New(testData.Type()).Interface()

			buf := bytes.NewReader(test.ExpectedBin)
			b.StartTimer()
			decoder := NewDecoder(buf)
			decoder.Decode(bval)
			b.StopTimer()
		}
	}
}

func TestDecode(t *testing.T) {
	var failed bool
	for _, test := range tests {
		if test.SkipDecode {
			continue
		}

		failed = false

		t.Logf("Testing Decode (%s)", test.Name)

		d := test.DecodeData
		if d == nil {
			d = test.Data
		}

		testData := reflect.ValueOf(test.Data)
		if !testData.IsValid() || isEmptyInterface(testData) {
			continue
		}
		if testData.Kind() == reflect.Ptr || testData.Kind() == reflect.Interface {
			testData = testData.Elem()
		}
		//typ := testData.Type()

		var err error
		var bval interface{}
		var xval interface{}
		var val interface{}

		if test.ExpectedBin != nil {
			bval = reflect.New(testData.Type()).Interface()
			buf := bytes.NewReader(test.ExpectedBin)
			decoder := NewDecoder(buf)
			err = decoder.Decode(bval)
			vt := reflect.ValueOf(bval)
			if vt.Kind() == reflect.Ptr || vt.Kind() == reflect.Interface {
				vt = vt.Elem()
				bval = vt.Interface()
			}
			val = bval
			if !reflect.DeepEqual(d, bval) {
				failed = true
			}
		}

		if !test.SkipDecodeXML && test.ExpectedXML != "" {
			xval = reflect.New(testData.Type()).Interface()
			buf := bytes.NewReader([]byte(test.ExpectedXML))
			decoder := NewDecoder(buf)
			err = decoder.Decode(xval)
			vt := reflect.ValueOf(xval)
			if vt.Kind() == reflect.Ptr || vt.Kind() == reflect.Interface {
				vt = vt.Elem()
				xval = vt.Interface()
			}
			val = xval
			if !reflect.DeepEqual(d, xval) {
				failed = true
			}
		}

		if bval != nil && xval != nil {
			if !reflect.DeepEqual(bval, xval) {
				t.Log("Binary and XML decoding yielded different values.")
				t.Log("Binary:", bval)
				t.Log("XML   :", xval)
				failed = true
			}
		}

		if failed {
			t.Log("Expected:", d)

			if err == nil {
				t.Log("Received:", val)
			} else {
				t.Log("   Error:", err)
			}
			t.Log("FAILED")
			t.Fail()
		}
	}
}

func ExampleDecoder_Decode() {
	type sparseBundleHeader struct {
		InfoDictionaryVersion string `plist:"CFBundleInfoDictionaryVersion"`
		BandSize              uint64 `plist:"band-size"`
		BackingStoreVersion   int    `plist:"bundle-backingstore-version"`
		DiskImageBundleType   string `plist:"diskimage-bundle-type"`
		Size                  uint64 `plist:"size"`
	}

	buf := bytes.NewReader([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
	<dict>
		<key>CFBundleInfoDictionaryVersion</key>
		<string>6.0</string>
		<key>band-size</key>
		<integer>8388608</integer>
		<key>bundle-backingstore-version</key>
		<integer>1</integer>
		<key>diskimage-bundle-type</key>
		<string>com.apple.diskimage.sparsebundle</string>
		<key>size</key>
		<integer>4398046511104</integer>
	</dict>
</plist>`))

	var data sparseBundleHeader
	decoder := NewDecoder(buf)
	err := decoder.Decode(&data)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(data)

	// Output: {6.0 8388608 1 com.apple.diskimage.sparsebundle 4398046511104}
}
