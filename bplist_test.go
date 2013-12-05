package plist

import (
	"bytes"
	"testing"
)

func BenchmarkBplistGenerate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		d := newBplistGenerator(nilWriter(0))
		d.generateDocument(plistValueTree)
	}
}

func BenchmarkBplistParse(b *testing.B) {
	buf := bytes.NewReader(plistValueTreeAsBplist)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		d := newBplistParser(buf)
		d.parseDocument()
		b.StopTimer()
		buf.Seek(0, 0)
	}
}
