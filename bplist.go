package plist

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"strconv"
	"time"
	"unicode/utf16"
)

type bplistTrailer struct {
	Unused            [5]uint8
	SortVersion       uint8
	OffsetIntSize     uint8
	ObjectRefSize     uint8
	NumObjects        uint64
	TopObject         uint64
	OffsetTableOffset uint64
}

const (
	bpTagNull        uint8 = 0x00
	bpTagBoolTrue          = 0x08
	bpTagBoolFalse         = 0x09
	bpTagInteger           = 0x10
	bpTagReal              = 0x20
	bpTagDate              = 0x30
	bpTagData              = 0x40
	bpTagASCIIString       = 0x50
	bpTagUTF16String       = 0x60
	bpTagArray             = 0xA0
	bpTagDictionary        = 0xD0
)

type bplistValueDecoder struct {
	reader   io.ReadSeeker
	version  int
	buf      []byte
	objrefs  map[uint64]*plistValue
	offtable []uint64
	trailer  bplistTrailer
}

func (p *bplistValueDecoder) decodeDocument() *plistValue {
	magic := make([]byte, 6)
	ver := make([]byte, 2)
	p.reader.Seek(0, 0)
	p.reader.Read(magic)
	if !bytes.Equal(magic, []byte("bplist")) {
		panic(errors.New("invalid binary property list (mismatched magic)"))
	}

	p.reader.Read(ver)
	if version, err := strconv.ParseInt(string(ver), 10, 0); err == nil {
		p.version = int(version)
	} else {
		panic(err)
	}

	p.objrefs = make(map[uint64]*plistValue)
	p.reader.Seek(-32, 2)
	binary.Read(p.reader, binary.BigEndian, &p.trailer)

	p.offtable = make([]uint64, p.trailer.NumObjects)

	// SEEK_SET
	p.reader.Seek(int64(p.trailer.OffsetTableOffset), 0)
	for i := uint64(0); i < p.trailer.NumObjects; i++ {
		off := p.readSizedInt(int(p.trailer.OffsetIntSize))
		p.offtable[i] = off
	}

	for _, off := range p.offtable {
		p.valueAtOffset(off)
	}

	return p.valueAtOffset(p.offtable[p.trailer.TopObject])
}

func (p *bplistValueDecoder) readSizedInt(nbytes int) uint64 {
	switch nbytes {
	case 1:
		var val uint8
		binary.Read(p.reader, binary.BigEndian, &val)
		return uint64(val)
	case 2:
		var val uint16
		binary.Read(p.reader, binary.BigEndian, &val)
		return uint64(val)
	case 4:
		var val uint32
		binary.Read(p.reader, binary.BigEndian, &val)
		return uint64(val)
	case 8:
		var val uint64
		binary.Read(p.reader, binary.BigEndian, &val)
		return uint64(val)
	}
	panic(errors.New("illegal integer size"))
}

func (p *bplistValueDecoder) countForTag(tag uint8) uint64 {
	cnt := uint64(tag & 0x0F)
	if cnt == 0xF {
		var intTag uint8
		binary.Read(p.reader, binary.BigEndian, &intTag)
		cnt = p.readSizedInt(1 << (intTag & 0xF))
	}
	return cnt
}

func (p *bplistValueDecoder) valueAtOffset(off uint64) *plistValue {
	if pval, ok := p.objrefs[off]; ok {
		return pval
	} else {
		pval := p.decodeTagAtOffset(int64(off))
		p.objrefs[off] = pval
		return pval
	}
	return nil
}

func (p *bplistValueDecoder) decodeTagAtOffset(off int64) *plistValue {
	var tag uint8
	p.reader.Seek(off, 0)
	binary.Read(p.reader, binary.BigEndian, &tag)

	switch tag & 0xF0 {
	case bpTagNull:
		switch tag & 0x0F {
		case bpTagBoolTrue, bpTagBoolFalse:
			return &plistValue{Boolean, tag == bpTagBoolTrue}
		}
	case bpTagInteger:
		val := p.readSizedInt(1 << (tag & 0xF))
		return &plistValue{Integer, val}
	case bpTagReal:
		nbytes := 1 << (tag & 0x0F)
		switch nbytes {
		case 4:
			var val float32
			binary.Read(p.reader, binary.BigEndian, &val)
			return &plistValue{Real, float64(val)}
		case 8:
			var val float64
			binary.Read(p.reader, binary.BigEndian, &val)
			return &plistValue{Real, float64(val)}
		}
		panic(errors.New("illegal float size"))
	case bpTagDate:
		var val float64
		binary.Read(p.reader, binary.BigEndian, &val)

		// Apple Epoch is 20110101000000Z
		// Adjust for UNIX Time
		val += 978307200

		sec, fsec := math.Modf(val)
		time := time.Unix(int64(sec), int64(fsec*float64(time.Second)))
		return &plistValue{Date, time}
	case bpTagData:
		cnt := p.countForTag(tag)

		bytes := make([]byte, cnt)
		binary.Read(p.reader, binary.BigEndian, bytes)
		return &plistValue{Data, bytes}
	case bpTagASCIIString, bpTagUTF16String:
		cnt := p.countForTag(tag)

		if tag&0xF0 == bpTagASCIIString {
			bytes := make([]byte, cnt)
			binary.Read(p.reader, binary.BigEndian, bytes)
			return &plistValue{String, string(bytes)}
		} else {
			bytes := make([]uint16, cnt)
			binary.Read(p.reader, binary.BigEndian, bytes)
			runes := utf16.Decode(bytes)
			return &plistValue{String, string(runes)}
		}
	case bpTagDictionary:
		cnt := p.countForTag(tag)

		dict := make(map[string]*plistValue)
		indices := make([]uint64, cnt*2)
		for i := uint64(0); i < cnt*2; i++ {
			idx := p.readSizedInt(int(p.trailer.ObjectRefSize))
			indices[i] = idx
		}
		for i := uint64(0); i < cnt; i++ {
			kval := p.valueAtOffset(p.offtable[indices[i]])
			dict[kval.value.(string)] = p.valueAtOffset(p.offtable[indices[i+cnt]])
		}

		return &plistValue{Dictionary, dict}
	case bpTagArray:
		cnt := p.countForTag(tag)

		arr := make([]*plistValue, cnt)
		indices := make([]uint64, cnt)
		for i := uint64(0); i < cnt; i++ {
			indices[i] = p.readSizedInt(int(p.trailer.ObjectRefSize))
		}
		for i := uint64(0); i < cnt; i++ {
			arr[i] = p.valueAtOffset(p.offtable[indices[i]])
		}

		return &plistValue{Array, arr}
	}
	return nil
}

func newBplistValueDecoder(r io.ReadSeeker) *bplistValueDecoder {
	return &bplistValueDecoder{reader: r}
}
