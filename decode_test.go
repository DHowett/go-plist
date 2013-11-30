package plist

import (
	"bytes"
	"reflect"
	"testing"
)

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
		var bval interface{} = reflect.New(testData.Type()).Interface()
		var xval interface{} = reflect.New(testData.Type()).Interface()
		var val interface{}

		if test.ExpectedBin != nil {
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

		if test.ExpectedXML != "" {
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
