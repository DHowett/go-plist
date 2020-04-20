package plist

import (
	"bytes"
	"testing"
)

type encodeOptionCase struct {
	Name     string
	Data     interface{}
	Options  []Option
	Validate func(error, []byte) bool
}

func validateThrowsError(err error, _ []byte) bool {
	return err != nil
}

var encodeOptionCases = []encodeOptionCase{
	{
		Name:    "Last Format Wins",
		Data:    uint64(1),
		Options: []Option{Format(XMLFormat), Format(OpenStepFormat)},
		Validate: func(err error, buf []byte) bool {
			return err == nil && buf[0] == '1'
		},
	},
	{
		Name:    "Indent",
		Data:    map[string]uint64{"A": 1},
		Options: []Option{Format(OpenStepFormat), Indent("*")},
		Validate: func(err error, buf []byte) bool {
			return err == nil && buf[2] == '*'
		},
	},
}

func TestEncoderOptions(t *testing.T) {
	for _, tc := range encodeOptionCases {
		subtest(t, tc.Name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			enc := newEncoderWithOptions(buf, tc.Options...)
			err := enc.Encode(tc.Data)
			if !tc.Validate(err, buf.Bytes()) {
				t.Fatalf("Failed validation; error <%v> buffer <%s>", err, buf.Bytes())
			}
		})
	}
}

func TestMarshalOptions(t *testing.T) {
	// runs the same cases through the marshaler
	for _, tc := range encodeOptionCases {
		subtest(t, tc.Name, func(t *testing.T) {
			buf, err := Marshal(tc.Data, AutomaticFormat, tc.Options...)
			if !tc.Validate(err, buf) {
				t.Fatalf("Failed validation; error <%v> buffer <%s>", err, buf)
			}
		})
	}
}
