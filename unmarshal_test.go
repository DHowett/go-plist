package plist

import (
	"reflect"
	"testing"
	"time"
)

func BenchmarkStructUnmarshal(b *testing.B) {
	type Data struct {
		Intarray []uint64  `plist:"intarray"`
		Floats   []float64 `plist:"floats"`
		Booleans []bool    `plist:"booleans"`
		Strings  []string  `plist:"strings"`
		Dat      []byte    `plist:"data"`
		Date     time.Time `plist:"date"`
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var xval Data
		d := &Decoder{}
		d.unmarshal(plistValueTree, reflect.ValueOf(&xval))
	}
}

func BenchmarkInterfaceUnmarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var xval interface{}
		d := &Decoder{}
		d.unmarshal(plistValueTree, reflect.ValueOf(&xval))
	}
}

func BenchmarkLargeArrayUnmarshal(b *testing.B) {
	var xval [1024]byte
	pval := cfData(make([]byte, 1024))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d := &Decoder{}
		d.unmarshal(pval, reflect.ValueOf(&xval))
	}
}

type CustomDate struct{}

func (cd *CustomDate) UnmarshalPlist(unmarshal func(interface{}) error) error { return nil }

func TestCustomDateUnmarshal(t *testing.T) {
	input := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
    <date>2003-02-03T09:00:00.00Z</date>
</plist>`

	var custom CustomDate
	if _, err := plist.Unmarshal([]byte(input), &custom); err != nil {
		t.Error(err)
	}
}
