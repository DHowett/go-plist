package plist

import (
	"reflect"
	"testing"
	"time"
)

var plistData *plistValue

func setupData() {
	plistData = &plistValue{
		Dictionary,
		map[string]*plistValue{
			"intarray": &plistValue{Array, []*plistValue{
				&plistValue{Integer, uint64(1)},
				&plistValue{Integer, uint64(8)},
				&plistValue{Integer, uint64(16)},
				&plistValue{Integer, uint64(32)},
				&plistValue{Integer, uint64(64)},
				&plistValue{Integer, uint64(2)},
				&plistValue{Integer, uint64(8)},
				&plistValue{Integer, uint64(17)},
				&plistValue{Integer, uint64(33)},
				&plistValue{Integer, uint64(65)},
			}},
			"floats": &plistValue{Array, []*plistValue{
				&plistValue{Real, sizedFloat{float64(32.0), 32}},
				&plistValue{Real, sizedFloat{float64(64.0), 64}},
			}},
			"booleans": &plistValue{Array, []*plistValue{
				&plistValue{Boolean, true},
				&plistValue{Boolean, false},
			}},
			"strings": &plistValue{Array, []*plistValue{
				&plistValue{String, "Hello, ASCII"},
				&plistValue{String, "Hello, 世界"},
			}},
			"data": &plistValue{Data, []byte{1, 2, 3, 4}},
			"date": &plistValue{Date, time.Date(2013, 11, 27, 0, 34, 0, 0, time.UTC)},
		},
	}
}

func BenchmarkStructTypeUnmarshal(b *testing.B) {
	type Data struct {
		Intarray []uint64  `plist:"intarray"`
		Floats   []float64 `plist:"floats"`
		Booleans []bool    `plist:"booleans"`
		Strings  []string  `plist:"strings"`
		Dat      []byte    `plist:"data"`
		Date     time.Time `plist:"date"`
	}
	setupData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var xval Data
		d := &Decoder{}
		d.unmarshal(plistData, reflect.ValueOf(&xval))
	}
}

func BenchmarkInterfaceTypeUnmarshal(b *testing.B) {
	setupData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var xval interface{}
		d := &Decoder{}
		d.unmarshal(plistData, reflect.ValueOf(&xval))
	}
}
