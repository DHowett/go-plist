package plist

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func BenchmarkXMLGenerate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		d := newXMLPlistGenerator(ioutil.Discard)
		d.generateDocument(plistValueTree)
	}
}

func BenchmarkXMLParse(b *testing.B) {
	buf := bytes.NewReader([]byte(plistValueTreeAsXML))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		d := newXMLPlistParser(buf)
		d.parseDocument()
		b.StopTimer()
		buf.Seek(0, 0)
	}
}
