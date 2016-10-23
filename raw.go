package plist

// RawPlistValue is a raw encoded Plist object. It implements Marshaler and
// Unmarshaler and can be used to delay Plist decoding or precompute a Plist
// encoding using DecodeElement or EncodeElement respectively.
type RawPlistValue plistValue

func (r *RawPlistValue) UnmarshalPlist(p *Decoder, start *RawPlistValue) error {
	*r = *start
	return nil
}

func (r RawPlistValue) MarshalPlist(p *Encoder, start *RawPlistValue) error {
	*start = r
	return nil
}
