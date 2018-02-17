// +build go1.4

package plist

import "encoding/xml"

func xmlInputOffset(d *xml.Decoder) int64 {
	return d.InputOffset()
}
