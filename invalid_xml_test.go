package plist

import (
	"bytes"
	"testing"
)

var InvalidXMLPlists = []struct {
	Name string
	Data string
}{
	{"Mismatched tag at root level", "<plist></dict>"},
	{"Mismatched tag in string", "<string>hello</world>"},
	{"Mismatched tag in dictionary", "<dict><key>key</key></what>"},
	{"Truncated integer", `<plist version="1.0"><integer>0x</integer></plist>`},
	{"Mismatched tag closing dict", "<plist><doct><key>helo</key><string></string></doct></plist>"},
	{"Dict without key", "<plist><dict><string>helo</string></dict></plist>"},
	{"Dict without value", "<plist><dict><key>helo</key></dict></plist>"},
	{"Empty plist", "<plist/>"},
	{"Empty integer", "<plist><integer></integer></plist>"},
	{"Empty real", "<plist><real></real></plist>"},
	{"Empty date", "<plist><date></date></plist>"},
	{"Unparseable integer", "<plist><integer>helo</integer></plist>"},
	{"Unparseable real", "<plist><real>helo</real></plist>"},
	{"Unparseable data", "<plist><data>*@&amp;%#helo</data></plist>"},
	{"Unparseable date", "<plist><date>*@&amp;%#helo</date></plist>"},
	{"Comment inside integer", "<plist><integer><!-- comment -->0</integer></plist>"},
	{"Comment inside real", "<plist><real><!-- comment -->0</real></plist>"},
	{"Comment inside data", "<plist><data><!-- comment -->0</data></plist>"},
	{"Comment inside date", "<plist><date><!-- comment -->0</date></plist>"},
	{"Comment inside string", "<plist><string><!-- comment -->0</string></plist>"},
	{"Mismatched tag closing string", "<plist><string></strong></plist>"},
	{"Unexpected directive in string", "<plist><string><!directive!></string></plist>"},
	{"Directive inside plist", "<plist><!ENTITY></plist>"},
	{"Directive inside array", "<plist><array><!ENTITY><true/></array></plist>"},
	{"Directive inside dictionary", "<plist><dict><!ENTITY><key/><true/></dict></plist>"},
	{"Unclosed integer", "<plist><integer>10</plist>"},
	{"Unclosed real", "<plist><real>10</plist>"},
	{"Unclosed string", "<plist><string>10</plist>"},
	{"Unclosed dict", "<plist><dict>10</plist>"},
	{"Unclosed dict key", "<plist><dict><key>10</plist>"},
	{"Unclosed plist", "<plist>"},
	{"Unclosed data", "<plist><data>"},
	{"Unclosed date", "<plist><date>"},
	{"Unclosed array", "<plist><array>"},
	{"Incomplete tag", "<pl"},
	{"Not an XML document", "bplist00"},
}

func TestInvalidXMLPlists(t *testing.T) {
	for _, test := range InvalidXMLPlists {
		subtest(t, test.Name, func(t *testing.T) {
			buf := bytes.NewReader([]byte(test.Data))
			d := newXMLPlistParser(buf)
			obj, err := d.parseDocument()
			if err == nil {
				t.Fatalf("invalid plist failed to throw error; deserialized %v", obj)
			} else {
				t.Log(err)
			}
		})
	}
}
