package plist

import (
	"testing"
)

func BenchmarkOpenStepGenerate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		d := &textPlistGenerator{nilWriter(0)}
		d.generateDocument(plistValueTree)
	}
}
