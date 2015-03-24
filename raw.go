package plist

// RawPlistValue is a raw encoded Plist object. It implements Marshaler and Unmarshaler and can be used to delay Plist decoding or precompute a Plist encoding.
type RawPlistValue plistValue

func (r *RawPlistValue) UnmarshalPlist(p *Decoder, start *plistValue) error {
	*r = RawPlistValue(*start)
	return nil
}

func (r RawPlistValue) MarshalPlist(p *Encoder, start *plistValue) error {
	*start = plistValue(r)
	return nil
}
