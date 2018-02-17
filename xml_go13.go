// +build !go1.4

package plist

import "encoding/xml"

func xmlInputOffset(*xml.Decoder) int64 {
	return -1
}
