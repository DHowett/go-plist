package plist

import (
	"strings"
	"testing"
)

var InvalidTextPlists []string = []string{
	"(/",
	"{/",
	"(/",
	"<*I>",
	"{0=(/",
	"(((/",
	"{0=/",
	"{0=((/",
	"/",
	"{0=((((/",
	"({/",
	"(<*I5>,<*I5>,<*I5>,<" +
		"*I5>,*I16777215>,<*I" +
		"268435455>,<*I429496" +
		"7295>,<*I18446744073" +
		"709551615>,)",
	"{0=(((/",
	"(<*I>",
	"<>",
	"((((/",
	"((/",
	"(<>",
}

func TestInvalidTextPlists(t *testing.T) {
	for _, data := range InvalidTextPlists {
		var obj interface{}
		buf := strings.NewReader(data)
		err := NewDecoder(buf).Decode(&obj)
		if err == nil {
			t.Fatal("invalid plist failed to throw error")
		} else {
			t.Log(err)
		}
	}
}
