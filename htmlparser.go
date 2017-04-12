package updatedetector

import (
	"bytes"
	"io"
	"strings"

	aelog "google.golang.org/appengine/log"

	"golang.org/x/net/context"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

var tagNameScript = []byte("script")
var tagNameStyle = []byte("style")

func HTMLToBytes(ctx context.Context, reader io.Reader, contentType string) []byte {
	var buf bytes.Buffer
	ereader, err := charset.NewReader(reader, contentType)
	if err != nil {
		aelog.Errorf(ctx, "%v", err)
		return buf.Bytes()
	}
	z := html.NewTokenizer(ereader)
	ignoreText := false
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			return buf.Bytes()
		case html.TextToken:
			if !ignoreText {
				text := strings.TrimSpace(string(z.Text()))
				if len(text) > 0 {
					buf.WriteString(text)
					buf.WriteString("\n")
				}
			}
		case html.StartTagToken:
			tn, _ := z.TagName()
			if bytes.Equal(tn, tagNameScript) {
				ignoreText = true
			} else if bytes.Equal(tn, tagNameStyle) {
				ignoreText = true
			}
		case html.EndTagToken:
			tn, _ := z.TagName()
			if bytes.Equal(tn, tagNameScript) {
				ignoreText = false
			} else if bytes.Equal(tn, tagNameStyle) {
				ignoreText = false
			}
		}
	}
}
