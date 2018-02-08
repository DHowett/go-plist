package cf

import (
	"hash/crc32"
	"sort"
	"time"
)

type Value interface {
	TypeName() string
	Hash() interface{}
}

type Dictionary struct {
	Keys   sort.StringSlice
	Values []Value
}

func (*Dictionary) TypeName() string {
	return "dictionary"
}

func (p *Dictionary) Hash() interface{} {
	return p
}

func (p *Dictionary) Len() int {
	return len(p.Keys)
}

func (p *Dictionary) Less(i, j int) bool {
	return p.Keys.Less(i, j)
}

func (p *Dictionary) Swap(i, j int) {
	p.Keys.Swap(i, j)
	p.Values[i], p.Values[j] = p.Values[j], p.Values[i]
}

func (p *Dictionary) Sort() {
	sort.Sort(p)
}

func (p *Dictionary) Range(r func(int, string, Value)) {
	p.Sort()
	for i, k := range p.Keys {
		r(i, k, p.Values[i])
	}
}

type Array struct {
	Values []Value
}

func (*Array) TypeName() string {
	return "array"
}

func (p *Array) Hash() interface{} {
	return p
}

func (p *Array) Range(r func(int, Value)) {
	for i, v := range p.Values {
		r(i, v)
	}
}

type String string

func (String) TypeName() string {
	return "string"
}

func (p String) Hash() interface{} {
	return string(p)
}

type Number struct {
	Signed bool
	Value  uint64
}

func (*Number) TypeName() string {
	return "integer"
}

func (p *Number) Hash() interface{} {
	if p.Signed {
		return int64(p.Value)
	}
	return p.Value
}

type Real struct {
	Wide  bool
	Value float64
}

func (Real) TypeName() string {
	return "real"
}

func (p *Real) Hash() interface{} {
	if p.Wide {
		return p.Value
	}
	return float32(p.Value)
}

type Boolean bool

func (Boolean) TypeName() string {
	return "boolean"
}

func (p Boolean) Hash() interface{} {
	return bool(p)
}

type UID uint64

func (UID) TypeName() string {
	return "UID"
}

func (p UID) Hash() interface{} {
	return p
}

type Data []byte

func (Data) TypeName() string {
	return "data"
}

func (p Data) Hash() interface{} {
	// Data are uniqued by their checksums.
	// Todo: Look at calculating this only once and storing it somewhere;
	// crc32 is fairly quick, however.
	return crc32.ChecksumIEEE([]byte(p))
}

type Date time.Time

func (Date) TypeName() string {
	return "date"
}

func (p Date) Hash() interface{} {
	return time.Time(p)
}
