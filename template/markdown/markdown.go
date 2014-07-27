// Package markdown implements a Markdown template converter.
//
// To enable it, import gnd.la/template/markdown like e.g.
//
//  import (
//	_ "gnd.la/template/markdown"
//  )
//
// Any templates with the .md extension will be converted to HTML
// interpreting their contents as Markdown.
// Since Go's template syntax needs to be escaping while using Markdown, any
// '{' or '}' character not inside a quoted or code block (delimited by
// either a single ` or three ```) will be automatically escaped.
// For example:
//
//  {{ fun .Foo .Bar }}
//
// Will be passed to Markdown  as:
//
//  \{\{ fun .Foo .Bar \}\}
//
// But the following won't be altered:
//
//  `{{ fun .Foo .Bar }}`
//
// Neither will:
//
//  ```
//  {{ fun .Foo .Bar }}
//  ```
//
package markdown

import (
	"bytes"
	"regexp"

	"gnd.la/template"

	"github.com/russross/blackfriday"
)

const (
	flags = blackfriday.HTML_USE_SMARTYPANTS | blackfriday.HTML_SMARTYPANTS_FRACTIONS |
		blackfriday.HTML_SMARTYPANTS_LATEX_DASHES
	//| blackfriday.HTML_SAFELINK - safe link breaks commands as link targets
	extensions = blackfriday.EXTENSION_NO_INTRA_EMPHASIS | blackfriday.EXTENSION_TABLES |
		blackfriday.EXTENSION_FENCED_CODE | blackfriday.EXTENSION_AUTOLINK |
		blackfriday.EXTENSION_STRIKETHROUGH | blackfriday.EXTENSION_SPACE_HEADERS
)

type unescape struct {
	escaped   []byte
	unescaped []byte
}

var (
	renderer     = blackfriday.HtmlRenderer(flags, "", "")
	beginComment = []byte("{{/*")
	endComment   = []byte("*/}}")

	// these need to be done in order, so
	// &amp; is the last one. Otherwise we would
	// also replace instances where the user
	// originally typed an escape sequence.
	unescapes = []*unescape{
		{[]byte("&quot;"), []byte("\"")},
		{[]byte("&ldquo;"), []byte("\"")},
		{[]byte("&rdquo;"), []byte("\"")},
		{[]byte("&gt;"), []byte(">")},
		{[]byte("&lt;"), []byte("<")},
		{[]byte("&amp;"), []byte("&")},
	}
	commandRe = regexp.MustCompile(`\{\{.*?\}\}`)
)

func toMarkdown(data []byte) ([]byte, error) {
	var out bytes.Buffer
	if bytes.HasPrefix(data, beginComment) {
		if end := bytes.Index(data, endComment); end >= 0 {
			prefix := data[:end+len(endComment)]
			out.Write(prefix)
			out.WriteByte('\n')
			data = data[end+len(endComment):]
		}
	}
	// Escape { and }, unless they're inside a quoted block
	var buf bytes.Buffer
	var quoted bool
	for ii := 0; ii < len(data); ii++ {
		v := data[ii]
		switch v {
		case '`':
			if ii < len(data)-2 && data[ii+1] == '`' && data[ii+2] == '`' {
				buf.WriteByte('`')
				buf.WriteByte('`')
				ii += 2
			}
			quoted = !quoted
			buf.WriteByte(v)
		case '{', '}':
			if !quoted {
				buf.WriteByte('\\')
			}
			fallthrough
		default:
			buf.WriteByte(v)
		}
	}
	md := blackfriday.Markdown(buf.Bytes(), renderer, extensions)
	// blackfriday escapes some characters inside commands, so we must
	// undo those escapes
	md = commandRe.ReplaceAllFunc(md, func(b []byte) []byte {
		for _, v := range unescapes {
			b = bytes.Replace(b, v.escaped, v.unescaped, -1)
		}
		return b
	})
	out.Write(md)
	return out.Bytes(), nil
}

func init() {
	template.RegisterConverter("md", toMarkdown)
}
