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

type bplistValueDecoder struct {
	reader   io.ReadSeeker
	version  int
	buf      []byte
	objrefs  map[uint64]*plistValue
	offtable []uint64
	trailer  struct {
		Unused            [5]uint8
		SortVersion       uint8
		OffsetIntSize     uint8
		ObjectRefSize     uint8
		NumObjects        uint64
		TopObject         uint64
		OffsetTableOffset uint64
	}
}

func (p *bplistValueDecoder) decodeDocument() (*plistValue, error) {
	magic := make([]byte, 6)
	ver := make([]byte, 2)
	p.reader.Seek(0, 0)
	p.reader.Read(magic)
	if !bytes.Equal(magic, []byte("bplist")) {
		return nil, errors.New("invalid binary property list (mismatched magic)")
	}

	p.reader.Read(ver)
	if version, err := strconv.ParseInt(string(ver), 10, 0); err == nil {
		p.version = int(version)
	} else {
		return nil, err
	}

	p.objrefs = make(map[uint64]*plistValue)
	p.reader.Seek(-32, 2)
	binary.Read(p.reader, binary.BigEndian, &p.trailer)

	p.offtable = make([]uint64, p.trailer.NumObjects)

	// SEEK_SET
	p.reader.Seek(int64(p.trailer.OffsetTableOffset), 0)
	for i := uint64(0); i < p.trailer.NumObjects; i++ {
		off, err := p.readIntBytes(int(p.trailer.OffsetIntSize))
		if err != nil {
			return nil, err
		}

		p.offtable[i] = off
	}

	for _, off := range p.offtable {
		p.valueAtOffset(off)
	}

	return p.valueAtOffset(p.offtable[p.trailer.TopObject])
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

func (p *bplistValueDecoder) readIntBytes(nbytes int) (uint64, error) {
	switch nbytes {
	case 1:
		var val uint8
		binary.Read(p.reader, binary.BigEndian, &val)
		return uint64(val), nil
	case 2:
		var val uint16
		binary.Read(p.reader, binary.BigEndian, &val)
		return uint64(val), nil
	case 4:
		var val uint32
		binary.Read(p.reader, binary.BigEndian, &val)
		return uint64(val), nil
	case 8:
		var val uint64
		binary.Read(p.reader, binary.BigEndian, &val)
		return uint64(val), nil
	}
	return 0, errors.New("illegal integer size")
}

func (p *bplistValueDecoder) valueAtOffset(off uint64) (*plistValue, error) {
	if pval, ok := p.objrefs[off]; ok {
		return pval, nil
	} else {
		pval, err := p.decodeTagAtOffset(int64(off))
		if err != nil {
			return nil, err
		}
		p.objrefs[off] = pval
		return pval, nil
	}
}

func (p *bplistValueDecoder) decodeTagAtOffset(off int64) (*plistValue, error) {
	var tag uint8
	p.reader.Seek(off, 0)
	binary.Read(p.reader, binary.BigEndian, &tag)

	switch tag & 0xF0 {
	case bpTagNull:
		switch tag & 0x0F {
		case bpTagBoolTrue, bpTagBoolFalse:
			return &plistValue{Boolean, tag == bpTagBoolTrue}, nil
		}
	case bpTagInteger:
		val, err := p.readIntBytes(1 << (tag & 0xF))
		if err != nil {
			return nil, err
		}
		return &plistValue{Integer, val}, nil
	case bpTagReal:
		nbytes := 1 << (tag & 0x0F)
		switch nbytes {
		case 4:
			var val float32
			binary.Read(p.reader, binary.BigEndian, &val)
			return &plistValue{Real, float64(val)}, nil
		case 8:
			var val float64
			binary.Read(p.reader, binary.BigEndian, &val)
			return &plistValue{Real, float64(val)}, nil
		}
		return nil, errors.New("illegal float size")
	case bpTagDate:
		var val float64
		binary.Read(p.reader, binary.BigEndian, &val)

		// Apple Epoch is 20110101000000Z
		// Adjust for UNIX Time
		val += 978307200

		sec, fsec := math.Modf(val)
		time := time.Unix(int64(sec), int64(fsec*float64(time.Second)))
		return &plistValue{Date, time}, nil
	case bpTagData:
		var nbytes uint64
		nbytes = uint64(tag & 0x0F)
		if nbytes == 0xF {
			var intTag uint8
			binary.Read(p.reader, binary.BigEndian, &intTag)

			var err error
			nbytes, err = p.readIntBytes(1 << (intTag & 0xF))
			if err != nil {
				return nil, err
			}
		}

		bytes := make([]byte, nbytes)
		binary.Read(p.reader, binary.BigEndian, bytes)
		return &plistValue{Data, bytes}, nil
	case bpTagASCIIString, bpTagUTF16String:
		var nchars uint64
		nchars = uint64(tag & 0x0F)
		if nchars == 0xF {
			var intTag uint8
			binary.Read(p.reader, binary.BigEndian, &intTag)

			var err error
			nchars, err = p.readIntBytes(1 << (intTag & 0xF))
			if err != nil {
				return nil, err
			}
		}

		if tag&0xF0 == bpTagASCIIString {
			bytes := make([]byte, nchars)
			binary.Read(p.reader, binary.BigEndian, bytes)
			return &plistValue{String, string(bytes)}, nil
		} else {
			bytes := make([]uint16, nchars)
			binary.Read(p.reader, binary.BigEndian, bytes)
			runes := utf16.Decode(bytes)
			return &plistValue{String, string(runes)}, nil
		}
	case bpTagDictionary:
		var nent uint64
		nent = uint64(tag & 0x0F)
		if nent == 0xF {
			var intTag uint8
			binary.Read(p.reader, binary.BigEndian, &intTag)

			var err error
			nent, err = p.readIntBytes(1 << (intTag & 0xF))
			if err != nil {
				return nil, err
			}
		}

		dict := make(map[string]*plistValue)
		indices := make([]uint64, nent*2)
		for i := uint64(0); i < nent*2; i++ {
			idx, err := p.readIntBytes(int(p.trailer.ObjectRefSize))
			if err != nil {
				return nil, err
			}
			indices[i] = idx
		}
		for i := uint64(0); i < nent; i++ {
			kval, err := p.valueAtOffset(p.offtable[indices[i]])
			if err != nil {
				return nil, err
			}

			pval, err := p.valueAtOffset(p.offtable[indices[i+nent]])
			if err != nil {
				return nil, err
			}
			dict[kval.value.(string)] = pval
		}

		return &plistValue{Dictionary, dict}, nil
	case bpTagArray:
		var nent uint64
		nent = uint64(tag & 0x0F)
		if nent == 0xF {
			var intTag uint8
			binary.Read(p.reader, binary.BigEndian, &intTag)

			var err error
			nent, err = p.readIntBytes(1 << (intTag & 0xF))
			if err != nil {
				return nil, err
			}
		}

		arr := make([]*plistValue, nent)
		indices := make([]uint64, nent)
		for i := uint64(0); i < nent; i++ {
			idx, err := p.readIntBytes(int(p.trailer.ObjectRefSize))
			if err != nil {
				return nil, err
			}
			indices[i] = idx
		}
		for i := uint64(0); i < nent; i++ {
			pval, err := p.valueAtOffset(p.offtable[indices[i]])
			if err != nil {
				return nil, err
			}
			arr[i] = pval
		}

		return &plistValue{Array, arr}, nil
	}
	return nil, nil
}

func newBplistValueDecoder(r io.ReadSeeker) *bplistValueDecoder {
	return &bplistValueDecoder{reader: r}
}
