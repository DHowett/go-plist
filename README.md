# plist - A pure Go property list transcoder
## INSTALL
	$ go get github.com/DHowett/go-plist

## FEATURES
* Supports encoding/decoding Apple property lists (both XML and binary) from/to arbitrary Go types

## USE
	package main
	import (
		"github.com/DHowett/go-plist"
		"os"
	)
	func main() {
		encoder := plist.NewEncoder(os.Stdout)
		encoder.Encode(map[string]string{"hello": "world"})
	}
