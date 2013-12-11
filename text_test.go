package plist

import (
	"bytes"
	"testing"
)

func BenchmarkOpenStepGenerate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		d := &textPlistGenerator{nilWriter(0)}
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
