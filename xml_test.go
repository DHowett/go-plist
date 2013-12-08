package plist

import (
	"bytes"
	"testing"
)

func BenchmarkXMLGenerate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		d := newXMLPlistGenerator(nilWriter(0))
		d.generateDocument(plistValueTree)
	}
}

func BenchmarkXMLParse(b *testing.B) {
	buf := bytes.NewReader([]byte(plistValueTreeAsXML))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		d := newXMLPlistParser(buf)
		d.parseDocument()
		b.StopTimer()
		buf.Seek(0, 0)
	}
}

func TestVariousIllegalXMLPlists(t *testing.T) {
	plists := []string{
		"<plist><doct><key>helo</key><string></string></doct></plist>",
		"<plist><dict><string>helo</string></dict></plist>",
		"<plist><dict><key>helo</key></dict></plist>",
		"<plist><integer>helo</integer></plist>",
		"<plist><real>helo</real></plist>",
		"<plist><data>*@&amp;%#helo</data></plist>",
		"<plist><date>*@&amp;%#helo</date></plist>",
		"<plist><date>*@&amp;%#helo</date></plist>",
		"<plist><integer>10</plist>",
		"<plist><real>10</plist>",
		"<plist><string>10</plist>",
		"<plist><dict>10</plist>",
		"<plist><dict><key>10</plist>",
		"<plist>",
		"<plist><data>",
		"<plist><date>",
		"<plist><array>",
		"<pl",
	}

	testDecode := func(plist string) (e error) {
		defer func() {
			if err := recover(); err != nil {
				e = err.(error)
			}
		}()
		buf := bytes.NewReader([]byte(plist))
		d := newXMLPlistParser(buf)
		d.parseDocument()
		return nil
	}

	for _, plist := range plists {
		err := testDecode(plist)
		t.Logf("Error: %v", err)
		if err == nil {
			t.Error("Expected error, received nothing.")
		}
	}
}
