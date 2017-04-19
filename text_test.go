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
			W, A, B, V, F, T, R, N, Hex1, Unicode1, Unicode2, Octal1 string
		}{
			"w", "\a", "\b", "\v", "\f", "\t", "\r", "\n", "\u00ab", "\u00ac", "\u00ad", "\033",
		},
		Expected: map[int][]byte{
			OpenStepFormat: []byte(`{
				W="\w";
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
	{
		Name: "Empty Data",
		Data: []byte{},
		Expected: map[int][]byte{
			OpenStepFormat: []byte(`<>`),
		},
	},
	{
		Name: "UTF-8 with BOM",
		Data: "Hello",
		Expected: map[int][]byte{
			OpenStepFormat: []byte("\uFEFFHello"),
		},
		SkipEncode: map[int]bool{OpenStepFormat: true},
	},
	{
		Name: "UTF-16LE with BOM",
		Data: "Hello",
		Expected: map[int][]byte{
			OpenStepFormat: []byte{0xFF, 0xFE, 'H', 0, 'e', 0, 'l', 0, 'l', 0, 'o', 0},
		},
		SkipEncode: map[int]bool{OpenStepFormat: true},
	},
	{
		Name: "UTF-16BE with BOM",
		Data: "Hello",
		Expected: map[int][]byte{
			OpenStepFormat: []byte{0xFE, 0xFF, 0, 'H', 0, 'e', 0, 'l', 0, 'l', 0, 'o'},
		},
		SkipEncode: map[int]bool{OpenStepFormat: true},
	},
	{
		Name: "UTF-16LE without BOM",
		Data: "Hello",
		Expected: map[int][]byte{
			OpenStepFormat: []byte{'H', 0, 'e', 0, 'l', 0, 'l', 0, 'o', 0},
		},
		SkipEncode: map[int]bool{OpenStepFormat: true},
	},
	{
		Name: "UTF-16BE without BOM",
		Data: "Hello",
		Expected: map[int][]byte{
			OpenStepFormat: []byte{0, 'H', 0, 'e', 0, 'l', 0, 'l', 0, 'o'},
		},
		SkipEncode: map[int]bool{OpenStepFormat: true},
	},
	{
		Name: "UTF-16BE with High Characters",
		Data: "Hello, 世界",
		Expected: map[int][]byte{
			OpenStepFormat: []byte{0, '"', 0, 'H', 0, 'e', 0, 'l', 0, 'l', 0, 'o', 0, ',', 0, ' ', 0x4E, 0x16, 0x75, 0x4C, 0, '"'},
		},
		SkipEncode: map[int]bool{OpenStepFormat: true},
	},
	{
		Name: "Legacy Strings File Format (No Dictionary)",
		Data: map[string]string{
			"Key":  "Value",
			"Key2": "Value2",
		},
		Expected: map[int][]byte{
			OpenStepFormat: []byte(`"Key" = "Value";
			"Key2" = "Value2";`),
		},
		SkipEncode: map[int]bool{OpenStepFormat: true},
	},
	{
		Name: "Strings File Shortcut Format (No Values)",
		Data: map[string]string{
			"Key":  "Key",
			"Key2": "Key2",
		},
		Expected: map[int][]byte{
			OpenStepFormat: []byte(`"Key";
			"Key2";`),
		},
		SkipEncode: map[int]bool{OpenStepFormat: true},
	},
	{
		Name: "Various Truncated Escapes",
		Data: "\x01\x02\x03\x04\x057",
		Expected: map[int][]byte{
			OpenStepFormat: []byte(`"\x1\u02\U003\4\0057"`),
		},
		SkipEncode: map[int]bool{OpenStepFormat: true},
	},
	{
		Name: "Various Case-Insensitive Escapes",
		Data: "\u00AB\uCDEF",
		Expected: map[int][]byte{
			OpenStepFormat: []byte(`"\xaB\uCdEf"`),
		},
		SkipEncode: map[int]bool{OpenStepFormat: true},
	},
	{
		Name: "Data long enough to trigger implementation-specific reallocation", // this is for coverage :(
		Data: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
		Expected: map[int][]byte{
			OpenStepFormat: []byte("<0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001>"),
		},
		SkipEncode: map[int]bool{OpenStepFormat: true},
	},
	{
		Name: "Empty Document",
		Data: map[string]interface{}{}, // Defined to be an empty dictionary
		Expected: map[int][]byte{
			OpenStepFormat: []byte{},
		},
		SkipEncode: map[int]bool{OpenStepFormat: true},
	},
	{
		Name: "Document consisting of only whitespace",
		Data: map[string]interface{}{}, // Defined to be an empty dictionary
		Expected: map[int][]byte{
			OpenStepFormat: []byte(" \n\t"),
		},
		SkipEncode: map[int]bool{OpenStepFormat: true},
	},
}

func TestTextDecode(t *testing.T) {
	for _, test := range textTestCases {
		t.Run(test.Name, func(t *testing.T) {
			actual := test.Data
			testData := reflect.ValueOf(actual)
			if !testData.IsValid() || isEmptyInterface(testData) {
				return
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
		})
	}
}
