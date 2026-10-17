package plist

import (
	"bytes"
	"io"
	"reflect"
	"runtime"
)

type peekReader struct {
	io.Reader
	io.Seeker
	peeked  []byte
	peekErr error
}

func newPeekReader(r io.Reader) *peekReader {
	s, _ := r.(io.Seeker)
	return &peekReader{r, s, nil, nil}
}

func (p *peekReader) Read(b []byte) (n int, err error) {
	if len(p.peeked) > 0 {
		// copy manages differing lengths properly
		n = copy(b, p.peeked)
		p.peeked = p.peeked[n:]
		if len(p.peeked) == 0 {
			// Upon consumption of all peeked bytes, vend the error we got from the parent reader
			err = p.peekErr
		}
		return
	}
	return p.Reader.Read(b)
}

func (p *peekReader) Peek(b []byte) (n int, err error) {
	n, err = p.Reader.Read(b)
	if p.Seeker != nil {
		p.Seeker.Seek(0, 0)
		return
		// Do not populate peeked; we can seek instead of replaying bytes
	}
	p.peeked = b[:n]
	p.peekErr = err
	return
}

func (p *peekReader) CanSeek() bool {
	return p.Seeker != nil
}

func headerMightBeXml(h []byte) bool {
	if bytes.IndexByte(h, '<') < 0 || bytes.Contains(h, []byte("<*")) {
		return false
	}
	// Any XML property list tags which cannot be confused for
	// hex-encoded NSData in the OpenStep format
	// (that is: if they look like hex, we need to check more characters)
	return bytes.Contains(h, []byte("<?")) ||
		bytes.Contains(h, []byte("<p")) ||
		bytes.Contains(h, []byte("<s")) ||
		bytes.Contains(h, []byte("<i")) ||
		bytes.Contains(h, []byte("<r")) ||
		bytes.Contains(h, []byte("<t")) ||
		bytes.Contains(h, []byte("<di")) ||
		bytes.Contains(h, []byte("<ar")) ||
		bytes.Contains(h, []byte("<fal")) ||
		bytes.Contains(h, []byte("<dat"))
}

type parser interface {
	parseDocument() (cfValue, error)
}

// A Decoder reads a property list from an input stream.
type Decoder struct {
	// the format of the most-recently-decoded property list
	Format int

	reader io.Reader
	lax    bool
}

// Decode works like Unmarshal, except it reads the decoder stream to find property list elements.
//
// After Decoding, the Decoder's Format field will be set to one of the plist format constants.
func (p *Decoder) Decode(v interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()

	r := newPeekReader(p.reader)

	header := make([]byte, 16)
	n, _ := r.Peek(header)

	var parser parser
	var pval cfValue
	if n >= 6 && bytes.Equal(header[:6], []byte("bplist")) {
		parser = newBplistParser(r)
		pval, err = parser.parseDocument()
		if err != nil {
			// Had a bplist header, but still got an error: we have to die here.
			return err
		}
		p.Format = BinaryFormat
	} else {
		tryXml := r.CanSeek()
		if !tryXml {
			// more expensive content checks only if we can't seek
			tryXml = headerMightBeXml(header)
		}

		if tryXml {
			parser = newXMLPlistParser(r)
			pval, err = parser.parseDocument()
		}

		// got an error parsing xml or didn't try xml
		if _, ok := err.(invalidPlistError); ok || !tryXml {
			if r.CanSeek() {
				// If we can seek after all, try.
				r.Seeker.Seek(0, 0)
			}

			// We don't use parser here because we want the textPlistParser type
			tp := newTextPlistParser(r)
			pval, err = tp.parseDocument()
			if err != nil {
				return err
			}
			p.Format = tp.format
			if p.Format == OpenStepFormat {
				// OpenStep property lists can only store strings,
				// so we have to turn on lax mode here for the unmarshal step later.
				p.lax = true
			}
		} else {
			if err != nil {
				return err
			}
			p.Format = XMLFormat
		}
	}

	p.unmarshal(pval, reflect.ValueOf(v))
	return
}

// NewDecoder returns a Decoder that reads property list elements from a stream reader, r.
// NewDecoder requires a Seekable stream for the purposes of file type detection.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{Format: InvalidFormat, reader: r, lax: false}
}

// Unmarshal parses a property list document and stores the result in the value pointed to by v.
//
// Unmarshal uses the inverse of the type encodings that Marshal uses, allocating heap-borne types as necessary.
//
// When given a nil pointer, Unmarshal allocates a new value for it to point to.
//
// To decode property list values into an interface value, Unmarshal decodes the property list into the concrete value contained
// in the interface value. If the interface value is nil, Unmarshal stores one of the following in the interface value:
//
//     string, bool, uint64, float64
//     plist.UID for "CoreFoundation Keyed Archiver UIDs" (convertible to uint64)
//     []byte, for plist data
//     []interface{}, for plist arrays
//     map[string]interface{}, for plist dictionaries
//
// If a property list value is not appropriate for a given value type, Unmarshal aborts immediately and returns an error.
//
// As Go does not support 128-bit types, and we don't want to pretend we're giving the user integer types (as opposed to
// secretly passing them structs), Unmarshal will drop the high 64 bits of any 128-bit integers encoded in binary property lists.
// (This is important because CoreFoundation serializes some large 64-bit values as 128-bit values with an empty high half.)
//
// When Unmarshal encounters an OpenStep property list, it will enter a relaxed parsing mode: OpenStep property lists can only store
// plain old data as strings, so we will attempt to recover integer, floating-point, boolean and date values wherever they are necessary.
// (for example, if Unmarshal attempts to unmarshal an OpenStep property list into a time.Time, it will try to parse the string it
// receives as a time.)
//
// Unmarshal returns the detected property list format and an error, if any.
func Unmarshal(data []byte, v interface{}) (format int, err error) {
	r := bytes.NewReader(data)
	dec := NewDecoder(r)
	err = dec.Decode(v)
	format = dec.Format
	return
}
