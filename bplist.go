package plist

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
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
	bpTagBoolFalse         = 0x08
	bpTagBoolTrue          = 0x09
	bpTagInteger           = 0x10
	bpTagReal              = 0x20
	bpTagDate              = 0x30
	bpTagData              = 0x40
	bpTagASCIIString       = 0x50
	bpTagUTF16String       = 0x60
	bpTagArray             = 0xA0
	bpTagDictionary        = 0xD0
)

type bplistValueEncoder struct {
	writer   *countedWriter
	uniqmap  map[interface{}]uint64
	objmap   map[*plistValue]uint64
	objtable []*plistValue
	nobjects uint64
	trailer  bplistTrailer
}

func (p *bplistValueEncoder) flattenPlistValue(pval *plistValue) {
	switch pval.kind {
	case String, Integer, Real, Date:
		if _, ok := p.uniqmap[pval.value]; ok {
			return
		}
		p.uniqmap[pval.value] = p.nobjects
	case Data:
		// Data are uniqued by their checksums.
		// The wonderful difference between uint64 (which we use for numbers)
		// and uint32 makes this possible.
		// Todo: Look at calculating this only once and storing it somewhere;
		// crc32 is fairly quick, however.
		uniqkey := crc32.ChecksumIEEE(pval.value.([]byte))
		if _, ok := p.uniqmap[uniqkey]; ok {
			return
		}
		p.uniqmap[uniqkey] = p.nobjects
	}

	p.objtable = append(p.objtable, pval)
	p.objmap[pval] = p.nobjects
	p.nobjects++

	switch pval.kind {
	case Dictionary:
		subvalues := pval.value.(map[string]*plistValue)
		for k, v := range subvalues {
			p.flattenPlistValue(&plistValue{String, k})
			p.flattenPlistValue(v)
		}
	case Array:
		subvalues := pval.value.([]*plistValue)
		for _, v := range subvalues {
			p.flattenPlistValue(v)
		}
	}
}

func (p *bplistValueEncoder) indexForPlistValue(pval *plistValue) (uint64, bool) {
	var v uint64
	var ok bool
	switch pval.kind {
	case String, Integer, Real, Date:
		v, ok = p.uniqmap[pval.value]
	case Data:
		v, ok = p.uniqmap[crc32.ChecksumIEEE(pval.value.([]byte))]
	default:
		v, ok = p.objmap[pval]
	}
	return v, ok
}

func (p *bplistValueEncoder) encodeDocument(rootpval *plistValue) {
	p.objtable = make([]*plistValue, 0, 15)
	p.uniqmap = make(map[interface{}]uint64)
	p.objmap = make(map[*plistValue]uint64)
	p.flattenPlistValue(rootpval)

	p.trailer.NumObjects = uint64(len(p.objtable))
	p.trailer.ObjectRefSize = uint8(minimumSizeForInt(p.trailer.NumObjects))

	p.writer.Write([]byte("bplist00"))

	offtable := make([]uint64, p.trailer.NumObjects)
	for i, pval := range p.objtable {
		offtable[i] = uint64(p.writer.BytesWritten())
		p.encodePlistValue(pval)
	}

	p.trailer.OffsetIntSize = uint8(minimumSizeForInt(uint64(p.writer.BytesWritten())))
	p.trailer.TopObject = p.objmap[rootpval]
	p.trailer.OffsetTableOffset = uint64(p.writer.BytesWritten())

	for _, offset := range offtable {
		p.writeSizedInt(offset, int(p.trailer.OffsetIntSize))
	}

	binary.Write(p.writer, binary.BigEndian, p.trailer)
}

func (p *bplistValueEncoder) encodePlistValue(pval *plistValue) {
	if pval == nil {
		return
	}

	switch pval.kind {
	case Dictionary:
		p.writeDictionaryTag(pval.value.(map[string]*plistValue))
	case Array:
		p.writeArrayTag(pval.value.([]*plistValue))
	case String:
		p.writeStringTag(pval.value.(string))
	case Integer:
		p.writeIntTag(pval.value.(uint64))
	case Real:
		p.writeRealTag(pval.value.(sizedFloat).value, pval.value.(sizedFloat).bits)
	case Boolean:
		p.writeBoolTag(pval.value.(bool))
	case Data:
		p.writeDataTag(pval.value.([]byte))
	case Date:
		p.writeDateTag(pval.value.(time.Time))
	}
}

func minimumSizeForInt(n uint64) int {
	switch {
	case n <= uint64(0xff):
		return 1
	case n <= uint64(0xffff):
		return 2
	case n <= uint64(0xffffffff):
		return 4
	default:
		return 8
	}
	panic(errors.New("illegal integer size"))
}

func (p *bplistValueEncoder) writeSizedInt(n uint64, nbytes int) {
	var val interface{}
	switch nbytes {
	case 1:
		val = uint8(n)
	case 2:
		val = uint16(n)
	case 4:
		val = uint32(n)
	case 8:
		val = n
	default:
		panic(errors.New("illegal integer size"))
	}
	err := binary.Write(p.writer, binary.BigEndian, val)
	if err != nil {
		panic(err)
	}
}

func (p *bplistValueEncoder) writeBoolTag(v bool) {
	tag := uint8(bpTagBoolFalse)
	if v {
		tag = bpTagBoolTrue
	}
	err := binary.Write(p.writer, binary.BigEndian, tag)
	if err != nil {
		panic(err)
	}
}

func (p *bplistValueEncoder) writeIntTag(n uint64) {
	var tag uint8
	var val interface{}
	switch {
	case n <= uint64(0xff):
		val = uint8(n)
		tag = bpTagInteger | 0x0
	case n <= uint64(0xffff):
		val = uint16(n)
		tag = bpTagInteger | 0x1
	case n <= uint64(0xffffffff):
		val = uint32(n)
		tag = bpTagInteger | 0x2
	default:
		val = n
		tag = bpTagInteger | 0x3
	}
	err := binary.Write(p.writer, binary.BigEndian, tag)
	if err != nil {
		panic(err)
	}

	err = binary.Write(p.writer, binary.BigEndian, val)
	if err != nil {
		panic(err)
	}
}

func (p *bplistValueEncoder) writeRealTag(n float64, bits int) {
	var tag uint8 = bpTagReal | 0x3
	var val interface{} = n
	if bits == 32 {
		val = float32(n)
		tag = bpTagReal | 0x2
	}
	err := binary.Write(p.writer, binary.BigEndian, tag)
	if err != nil {
		panic(err)
	}

	err = binary.Write(p.writer, binary.BigEndian, val)
	if err != nil {
		panic(err)
	}
}

func (p *bplistValueEncoder) writeDateTag(t time.Time) {
	tag := uint8(bpTagDate) | 0x3
	val := float64(t.In(time.UTC).UnixNano()) / float64(time.Second)
	val -= 978307200 // Adjust to Apple Epoch
	err := binary.Write(p.writer, binary.BigEndian, tag)
	if err != nil {
		panic(err)
	}

	err = binary.Write(p.writer, binary.BigEndian, val)
	if err != nil {
		panic(err)
	}
}

func (p *bplistValueEncoder) writeCountedTag(tag uint8, count uint64) {
	marker := tag
	if count > 0xF {
		marker |= 0xF
	} else {
		marker |= uint8(count)
	}

	err := binary.Write(p.writer, binary.BigEndian, marker)
	if err != nil {
		panic(err)
	}

	if count > 0xF {
		p.writeIntTag(count)
	}
}

func (p *bplistValueEncoder) writeDataTag(data []byte) {
	p.writeCountedTag(bpTagData, uint64(len(data)))
	err := binary.Write(p.writer, binary.BigEndian, data)
	if err != nil {
		panic(err)
	}
}

func (p *bplistValueEncoder) writeStringTag(str string) {
	var err error
	for _, r := range str {
		if r > 0xFF {
			utf16Runes := utf16.Encode([]rune(str))
			p.writeCountedTag(bpTagUTF16String, uint64(len(utf16Runes)))
			err = binary.Write(p.writer, binary.BigEndian, utf16Runes)
			return
		}
	}

	p.writeCountedTag(bpTagASCIIString, uint64(len(str)))
	err = binary.Write(p.writer, binary.BigEndian, []byte(str))

	if err != nil {
		panic(err)
	}
}

func (p *bplistValueEncoder) writeDictionaryTag(dict map[string]*plistValue) {
	p.writeCountedTag(bpTagDictionary, uint64(len(dict)))
	vals := make([]uint64, len(dict)*2)
	cnt := len(dict)
	i := 0
	for k, v := range dict {
		keyIdx, ok := p.uniqmap[k]
		if !ok {
			panic(errors.New("failed to find key " + k + " in object map during serialization"))
		}

		objIdx, ok := p.indexForPlistValue(v)
		if !ok {
			panic(errors.New("failed to find value for key " + k + " in object map during serialization"))
		}

		vals[i] = keyIdx
		vals[i+cnt] = objIdx
		i++
	}

	for _, v := range vals {
		p.writeSizedInt(v, int(p.trailer.ObjectRefSize))
	}
}

func (p *bplistValueEncoder) writeArrayTag(arr []*plistValue) {
	p.writeCountedTag(bpTagArray, uint64(len(arr)))
	for _, v := range arr {
		objIdx, ok := p.indexForPlistValue(v)
		if !ok {
			panic(errors.New("failed to find value in object map during serialization"))
		}

		p.writeSizedInt(objIdx, int(p.trailer.ObjectRefSize))
	}
}

func newBplistValueEncoder(w io.Writer) *bplistValueEncoder {
	return &bplistValueEncoder{
		writer: &countedWriter{Writer: w},
	}
}

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
