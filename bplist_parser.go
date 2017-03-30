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
	objects       []cfValue // object ID to object
	offtable      []uint64
	trailer       bplistTrailer
	trailerOffset int64

	containerStack []uint64 // slice of object IDs; manipulated during container deserialization
}

func (p *bplistParser) validateObjectListLength(off int64, oid uint64, length uint64, context string) {
	if uint64(off)+(length*uint64(p.trailer.ObjectRefSize)) > p.trailer.OffsetTableOffset {
		panic(fmt.Errorf("%s#%d length (%v) puts its end beyond the offset table at 0x%x", context, oid, length, p.trailer.OffsetTableOffset))
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
		panic(errors.New("binary property list offset size isn't big enough to address entire file"))
	}

	if p.trailer.TopObject >= p.trailer.NumObjects {
		panic(fmt.Errorf("top object #%d is out of range (only %d objects exist)", p.trailer.TopObject, p.trailer.NumObjects))
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

	must2(p.reader.Read(ver))

	p.version = int(mustParseInt(string(ver), 10, 0))

	if p.version > 1 {
		panic(fmt.Errorf("unexpected version %d", p.version))
	}

	var err error
	p.trailerOffset, err = p.reader.Seek(-32, 2)
	if err != nil {
		panic(err)
	}

	must(binary.Read(p.reader, binary.BigEndian, &p.trailer))
	p.validateDocumentTrailer()

	// INVARIANTS:
	// - Entire offset table is before trailer
	// - Offset table begins after header
	// - Offset table can address entire file
	// - Object IDs are big enough to support the number of objects in this plist
	// - Top object is in range

	must2(p.reader.Seek(int64(p.trailer.OffsetTableOffset), 0)) // SEEK_SET

	p.objects = make([]cfValue, p.trailer.NumObjects)
	p.offtable = make([]uint64, p.trailer.NumObjects)
	maxOffset := p.trailer.OffsetTableOffset - 1
	for i := uint64(0); i < p.trailer.NumObjects; i++ {
		off, _ := p.readSizedInt(int(p.trailer.OffsetIntSize))
		if off > maxOffset {
			panic(fmt.Errorf("object#%d starts beyond beginning of object table (0x%x, table@0x%x)", i, off, maxOffset+1))
		}
		p.offtable[i] = off
	}

	root := p.objectAtIndex(p.trailer.TopObject)

	pval = root
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

func (p *bplistParser) objectAtIndex(index uint64) cfValue {
	if index >= p.trailer.NumObjects {
		panic(fmt.Errorf("invalid object #%d (max %d)", index, p.trailer.NumObjects))
	}

	if pval := p.objects[index]; pval != nil {
		return pval
	}
	pval := p.parseTagAtOffset(int64(p.offtable[index]), index)
	p.objects[index] = pval
	return pval

}

func (p *bplistParser) panicNestedObject(oid uint64) {
	oids := ""
	for _, v := range p.containerStack {
		oids += fmt.Sprintf("#%d > ", v)
	}

	// %s%d: oids above ends with " > "
	panic(fmt.Errorf("self-referential collection#%d (%s#%d) cannot be deserialized", oid, oids, oid))
}

func (p *bplistParser) parseTagAtOffset(off int64, oid uint64) cfValue {
	for _, v := range p.containerStack {
		if v == oid {
			p.panicNestedObject(oid)
		}
	}
	p.containerStack = append(p.containerStack, oid)
	defer func() {
		p.containerStack = p.containerStack[:len(p.containerStack)-1]
	}()

	var tag uint8
	must2(p.reader.Seek(off, 0))
	must(binary.Read(p.reader, binary.BigEndian, &tag))

	switch tag & 0xF0 {
	case bpTagNull:
		switch tag & 0x0F {
		case bpTagBoolTrue, bpTagBoolFalse:
			return cfBoolean(tag == bpTagBoolTrue)
		}
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
			panic(fmt.Errorf("data#%d @ %x longer than file (%v bytes, max is %v)", oid, off, cnt, p.trailer.OffsetTableOffset))
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
			panic(fmt.Errorf("string#%d @ %x longer than file (%v bytes, max is %v)", oid, off, cnt*characterWidth, p.trailer.OffsetTableOffset))
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
		p.validateObjectListLength(off, oid, cnt*2, "dictionary")

		keys := make([]string, cnt)
		values := make([]cfValue, cnt)
		indices := make([]uint64, cnt*2)
		for i := uint64(0); i < cnt*2; i++ {
			indices[i], _ = p.readSizedInt(int(p.trailer.ObjectRefSize))
		}

		for i := uint64(0); i < cnt; i++ {
			kval := p.objectAtIndex(indices[i])
			vval := p.objectAtIndex(indices[i+cnt])

			if str, ok := kval.(cfString); ok {
				keys[i] = string(str)
				values[i] = vval
			} else {
				panic(fmt.Errorf("dictionary contains non-string key at index %d", i))
			}
		}

		return &cfDictionary{keys: keys, values: values}
	case bpTagArray:
		cnt := p.countForTag(tag)
		p.validateObjectListLength(off, oid, cnt, "array")

		// this is fully read in advance because objectAtIndex can seek.
		indices := make([]uint64, cnt)
		for i := uint64(0); i < cnt; i++ {
			indices[i], _ = p.readSizedInt(int(p.trailer.ObjectRefSize))
		}

		arr := make([]cfValue, cnt)
		for i, newOid := range indices {
			arr[i] = p.objectAtIndex(newOid)
		}

		return &cfArray{arr}
	}
	panic(fmt.Errorf("unexpected atom#%d 0x%2.02x at offset %d", oid, tag, off))
}

func newBplistParser(r io.ReadSeeker) *bplistParser {
	return &bplistParser{reader: r}
}
