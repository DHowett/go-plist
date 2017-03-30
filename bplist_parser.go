package plist

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"runtime"
	"time"
	"unicode/utf16"
)

type bplistParser struct {
	reader        io.ReadSeeker
	version       int
	objrefs       map[uint64]cfValue
	offtable      []uint64
	trailer       bplistTrailer
	trailerOffset int64

	delayedObjects map[*cfValue]uint64

	containerStack []uint64 // slice of object IDs; manipulated during container deserialization
}

func (p *bplistParser) validateObjectListLength(off int64, length uint64, context string) {
	if uint64(off)+(length*uint64(p.trailer.ObjectRefSize)) > p.trailer.OffsetTableOffset {
		panic(fmt.Errorf("%s length (%v) puts its end beyond the offset table at 0x%x", context, length, p.trailer.OffsetTableOffset))
	}
}

func (p *bplistParser) validateDocumentTrailer() {
	if p.trailer.OffsetTableOffset >= uint64(p.trailerOffset) {
		panic(fmt.Errorf("binary property list offset table beyond beginning of trailer (0x%x, trailer@0x%x)", p.trailer.OffsetTableOffset, p.trailerOffset))
	}

	if p.trailer.OffsetTableOffset < 9 {
		panic(fmt.Errorf("binary property list offset table begins inside header (0x%x)", p.trailer.OffsetTableOffset))
	}

	if uint64(p.trailerOffset) > (p.trailer.NumObjects*uint64(p.trailer.OffsetIntSize))+p.trailer.OffsetTableOffset {
		panic(errors.New("binary property list contains garbage between offset table and trailer"))
	}

	if p.trailer.NumObjects > uint64(p.trailerOffset) {
		panic(fmt.Errorf("binary property list contains more objects (%v) than there are non-trailer bytes in the file (%v)", p.trailer.NumObjects, p.trailerOffset))
	}

	objectRefSize := uint64(1) << (8 * p.trailer.ObjectRefSize)
	if p.trailer.NumObjects > objectRefSize {
		panic(fmt.Errorf("binary property list contains more objects (%v) than its object ref size (%v bytes) can support", p.trailer.NumObjects, p.trailer.ObjectRefSize))
	}

	if p.trailer.OffsetIntSize < uint8(8) && (uint64(1)<<(8*p.trailer.OffsetIntSize)) <= p.trailer.OffsetTableOffset {
		panic(errors.New("binary property offset size isn't big enough to address entire file"))
	}

	if p.trailer.TopObject >= p.trailer.NumObjects {
		panic(fmt.Errorf("top object index %v is out of range (only %v objects exist)", p.trailer.TopObject, p.trailer.NumObjects))
	}
}

func (p *bplistParser) parseDocument() (pval cfValue, parseError error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			if _, ok := r.(invalidPlistError); ok {
				parseError = r.(error)
			} else {
				// Wrap all non-invalid-plist errors.
				parseError = plistParseError{"binary", r.(error)}
			}
		}
	}()

	magic := make([]byte, 6)
	ver := make([]byte, 2)
	p.reader.Seek(0, 0)
	p.reader.Read(magic)
	if !bytes.Equal(magic, []byte("bplist")) {
		panic(invalidPlistError{"binary", errors.New("mismatched magic")})
	}

	_, err := p.reader.Read(ver)
	if err != nil {
		panic(err)
	}

	p.version = int(mustParseInt(string(ver), 10, 0))

	if p.version > 1 {
		panic(fmt.Errorf("unexpected version %d", p.version))
	}

	p.objrefs = make(map[uint64]cfValue)
	p.trailerOffset, err = p.reader.Seek(-32, 2)
	if err != nil && err != io.EOF {
		panic(err)
	}

	err = binary.Read(p.reader, binary.BigEndian, &p.trailer)
	if err != nil && err != io.EOF {
		panic(err)
	}

	p.validateDocumentTrailer()
	p.offtable = make([]uint64, p.trailer.NumObjects)

	// SEEK_SET
	_, err = p.reader.Seek(int64(p.trailer.OffsetTableOffset), 0)
	if err != nil && err != io.EOF {
		panic(err)
	}

	maxOffset := p.trailer.OffsetTableOffset - 1
	for i := uint64(0); i < p.trailer.NumObjects; i++ {
		off, _ := p.readSizedInt(int(p.trailer.OffsetIntSize))
		if off > maxOffset {
			panic(fmt.Errorf("object %v starts beyond beginning of object table (0x%x, table@0x%x)", i, off, maxOffset+1))
		}
		p.offtable[i] = off
	}

	p.delayedObjects = make(map[*cfValue]uint64)

	for _, off := range p.offtable {
		p.valueAtOffset(off)
	}

	for pvalp, off := range p.delayedObjects {
		if pval, ok := p.objrefs[off]; ok {
			*pvalp = pval
		} else {
			panic(fmt.Errorf("object@0x%x not referenced by object table", off))
		}
	}

	pval = p.valueAtOffset(p.offtable[p.trailer.TopObject])
	return
}

// readSizedInt returns a 128-bit integer as low64, high64
func (p *bplistParser) readSizedInt(nbytes int) (uint64, uint64) {
	switch nbytes {
	case 1:
		var val uint8
		binary.Read(p.reader, binary.BigEndian, &val)
		return uint64(val), 0
	case 2:
		var val uint16
		binary.Read(p.reader, binary.BigEndian, &val)
		return uint64(val), 0
	case 4:
		var val uint32
		binary.Read(p.reader, binary.BigEndian, &val)
		return uint64(val), 0
	case 8:
		var val uint64
		binary.Read(p.reader, binary.BigEndian, &val)
		return val, 0
	case 16:
		var high, low uint64
		binary.Read(p.reader, binary.BigEndian, &high)
		binary.Read(p.reader, binary.BigEndian, &low)
		// TODO: int128 support (!)
		return low, high
	}
	panic(errors.New("illegal integer size"))
}

func (p *bplistParser) countForTag(tag uint8) uint64 {
	cnt := uint64(tag & 0x0F)
	if cnt == 0xF {
		var intTag uint8
		binary.Read(p.reader, binary.BigEndian, &intTag)
		cnt, _ = p.readSizedInt(1 << (intTag & 0xF))
	}
	return cnt
}

func (p *bplistParser) valueAtOffset(off uint64) cfValue {
	if pval, ok := p.objrefs[off]; ok {
		return pval
	}
	pval := p.parseTagAtOffset(int64(off))
	p.objrefs[off] = pval
	return pval
}

func (p *bplistParser) parseTagAtOffset(off int64) cfValue {
	var tag uint8
	_, err := p.reader.Seek(off, 0)
	if err != nil {
		panic(err)
	}
	err = binary.Read(p.reader, binary.BigEndian, &tag)
	if err != nil {
		panic(err)
	}

	switch tag & 0xF0 {
	case bpTagNull:
		switch tag & 0x0F {
		case bpTagBoolTrue, bpTagBoolFalse:
			return cfBoolean(tag == bpTagBoolTrue)
		}
		return nil
	case bpTagInteger:
		lo, hi := p.readSizedInt(1 << (tag & 0xF))
		return &cfNumber{
			signed: hi == 0xFFFFFFFFFFFFFFFF, // a signed integer is stored as a 128-bit integer with the top 64 bits set
			value:  lo,
		}
	case bpTagReal:
		nbytes := 1 << (tag & 0x0F)
		switch nbytes {
		case 4:
			var val float32
			binary.Read(p.reader, binary.BigEndian, &val)
			return &cfReal{wide: false, value: float64(val)}
		case 8:
			var val float64
			binary.Read(p.reader, binary.BigEndian, &val)
			return &cfReal{wide: true, value: val}
		}
		panic(errors.New("illegal float size"))
	case bpTagDate:
		var val float64
		binary.Read(p.reader, binary.BigEndian, &val)

		// Apple Epoch is 20110101000000Z
		// Adjust for UNIX Time
		val += 978307200

		sec, fsec := math.Modf(val)
		time := time.Unix(int64(sec), int64(fsec*float64(time.Second))).In(time.UTC)
		return cfDate(time)
	case bpTagData:
		cnt := p.countForTag(tag)
		if uint64(off+int64(cnt)) > p.trailer.OffsetTableOffset {
			panic(fmt.Errorf("data at %x longer than file (%v bytes, max is %v)", off, cnt, p.trailer.OffsetTableOffset))
		}

		bytes := make([]byte, cnt)
		binary.Read(p.reader, binary.BigEndian, bytes)
		return cfData(bytes)
	case bpTagASCIIString, bpTagUTF16String:
		cnt := p.countForTag(tag)
		characterWidth := uint64(1)
		if tag&0xF0 == bpTagUTF16String {
			characterWidth = 2
		}
		if uint64(off+int64(cnt*characterWidth)) > p.trailer.OffsetTableOffset {
			panic(fmt.Errorf("string at %x longer than file (%v bytes, max is %v)", off, cnt*characterWidth, p.trailer.OffsetTableOffset))
		}

		if tag&0xF0 == bpTagASCIIString {
			bytes := make([]byte, cnt)
			binary.Read(p.reader, binary.BigEndian, bytes)
			return cfString(bytes)
		}

		bytes := make([]uint16, cnt)
		binary.Read(p.reader, binary.BigEndian, bytes)
		runes := utf16.Decode(bytes)
		return cfString(runes)
	case bpTagUID: // Somehow different than int: low half is nbytes - 1 instead of log2(nbytes)
		val, _ := p.readSizedInt(int(tag&0xF) + 1)
		return cfUID(val)
	case bpTagDictionary:
		cnt := p.countForTag(tag)
		p.validateObjectListLength(off, cnt*2, "dictionary")

		keys := make([]string, cnt)
		values := make([]cfValue, cnt)
		indices := make([]uint64, cnt*2)
		for i := uint64(0); i < cnt*2; i++ {
			idx, _ := p.readSizedInt(int(p.trailer.ObjectRefSize))

			if idx >= p.trailer.NumObjects {
				panic(fmt.Errorf("dictionary contains invalid entry index %d (max %d)", idx, p.trailer.NumObjects))
			}

			indices[i] = idx
		}

		for i := uint64(0); i < cnt; i++ {
			keyOffset := p.offtable[indices[i]]
			valueOffset := p.offtable[indices[i+cnt]]
			if keyOffset == uint64(off) {
				panic(fmt.Errorf("dictionary contains self-referential key %x (index %d)", off, i))
			}
			if valueOffset == uint64(off) {
				panic(fmt.Errorf("dictionary contains self-referential value %x (index %d)", off, i))
			}

			kval := p.valueAtOffset(keyOffset)
			if str, ok := kval.(cfString); ok {
				keys[i] = string(str)
				p.delayedObjects[&values[i]] = valueOffset
			} else {
				panic(fmt.Errorf("dictionary contains non-string key at index %d", i))
			}
		}

		return &cfDictionary{keys: keys, values: values}
	case bpTagArray:
		cnt := p.countForTag(tag)
		p.validateObjectListLength(off, cnt, "array")

		arr := make([]cfValue, cnt)
		indices := make([]uint64, cnt)
		for i := uint64(0); i < cnt; i++ {
			idx, _ := p.readSizedInt(int(p.trailer.ObjectRefSize))

			if idx >= p.trailer.NumObjects {
				panic(fmt.Errorf("array contains invalid entry index %d (max %d)", idx, p.trailer.NumObjects))
			}

			indices[i] = idx
		}
		for i := uint64(0); i < cnt; i++ {
			valueOffset := p.offtable[indices[i]]
			if valueOffset == uint64(off) {
				panic(fmt.Errorf("array contains self-referential value %x (index %d)", off, i))
			}
			p.delayedObjects[&arr[i]] = valueOffset
		}

		return &cfArray{arr}
	}
	panic(fmt.Errorf("unexpected atom 0x%2.02x at offset %d", tag, off))
}

func newBplistParser(r io.ReadSeeker) *bplistParser {
	return &bplistParser{reader: r}
}
