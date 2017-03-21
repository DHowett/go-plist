package plist

import (
	"strings"
	"testing"
)

var InvalidTextPlists = []string{
	"(/",
	"{/",
	"<*I>",
	"{0=(/",
	"(((/",
	"{0=/",
	"{0=((/",
	"/",
	"{0=((((/",
	"({/",
	"(<*I5>,<*I5>,<*I5>,<*I5>,*I16777215>,<*I268435455>,<*I4294967295>,<*I18446744073709551615>,)",
	"{0=(((/",
	"(<*I>",
	"<>",
	"((((/",
	"((/",
	"(<>",
	"{¬=A;}",  // that character should be in quotes for goth GNUStep and OpenStep
	`{"A"A;}`, // there should be an = between "A" and A
	`{"A"=A}`, // there should be a ; at the end of the dictionary
	"<*F33>",  // invalid GNUstep type F
	"<EQ>",    // invalid data that isn't hex
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
