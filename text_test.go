package plist

import (
	"bytes"
	"io/ioutil"
	"reflect"
	"testing"
)

func BenchmarkOpenStepGenerate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		d := newTextPlistGenerator(ioutil.Discard, OpenStepFormat)
		d.generateDocument(plistValueTree)
	}
}

func BenchmarkOpenStepParse(b *testing.B) {
	buf := bytes.NewReader([]byte(plistValueTreeAsOpenStep))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		d := newTextPlistParser(buf)
		d.parseDocument()
		b.StopTimer()
		buf.Seek(0, 0)
	}
}

func BenchmarkGNUStepParse(b *testing.B) {
	buf := bytes.NewReader([]byte(plistValueTreeAsGNUStep))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		d := newTextPlistParser(buf)
		d.parseDocument()
		b.StopTimer()
		buf.Seek(0, 0)
	}
}

var textTestCases = []TestData{
	{
		Name: "Comments",
		Data: struct {
			A, B, C int
			S, S2   string
		}{
			1, 2, 3,
			"/not/a/comment/", "/not*a/*comm*en/t",
		},
		Expected: map[int][]byte{
			OpenStepFormat: []byte(`{
				A=1 /* A is 1 because it is the first letter */;
				B=2; // B is 2 because comment-to-end-of-line.
				C=3;
				S = /not/a/comment/;
				S2 = /not*a/*comm*en/t;
			}`),
		},
	},
	{
		Name: "Escapes",
		Data: struct {
			A, B, V, F, T, R, N, Hex1, Unicode1, Unicode2, Octal1 string
		}{
			"\a", "\b", "\v", "\f", "\t", "\r", "\n", "\u00ab", "\u00ac", "\u00ad", "\033",
		},
		Expected: map[int][]byte{
			OpenStepFormat: []byte(`{
				A="\a";
				B="\b";
				V="\v";
				F="\f";
				T="\t";
				R="\r";
				N="\n";
				Hex1="\xAB";
				Unicode1="\u00AC";
				Unicode2="\U00AD";
				Octal1="\033";
			}`),
		},
	},
	{
		Name: "Empty Strings in Arrays",
		Data: []string{"A"},
		Expected: map[int][]byte{
			OpenStepFormat: []byte(`(A,,,"",)`),
		},
	},
}

func TestTextDecode(t *testing.T) {
	for _, test := range textTestCases {
		actual := test.Data
		testData := reflect.ValueOf(actual)
		if !testData.IsValid() || isEmptyInterface(testData) {
			continue
		}
		if testData.Kind() == reflect.Ptr || testData.Kind() == reflect.Interface {
			testData = testData.Elem()
		}
		actual = testData.Interface()

		parsed := reflect.New(testData.Type()).Interface()
		buf := bytes.NewReader(test.Expected[OpenStepFormat])
		decoder := NewDecoder(buf)
		err := decoder.Decode(parsed)
		if err != nil {
			t.Error(err.Error())
		}

		vt := reflect.ValueOf(parsed)
		if vt.Kind() == reflect.Ptr || vt.Kind() == reflect.Interface {
			vt = vt.Elem()
			parsed = vt.Interface()
		}

		if !reflect.DeepEqual(actual, parsed) {
			t.Logf("Expected: %#v", actual)
			t.Logf("Received: %#v", parsed)
			t.Fail()
		}
	}
}
