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
// Since Go's template syntax might conflict with Markdown's, any
// commands (i.e. blocks between {{ and }}) won't be processed with
// Markdown and will be included verbatim in the resulting template.
package markdown

import (
	"bytes"
	"fmt"
	"github.com/russross/blackfriday"
	"gnd.la/template"
	"regexp"
	"strconv"
)

const (
	flags = blackfriday.HTML_USE_SMARTYPANTS | blackfriday.HTML_SMARTYPANTS_FRACTIONS |
		blackfriday.HTML_SMARTYPANTS_LATEX_DASHES | blackfriday.HTML_SAFELINK
	extensions = blackfriday.EXTENSION_NO_INTRA_EMPHASIS | blackfriday.EXTENSION_TABLES |
		blackfriday.EXTENSION_FENCED_CODE | blackfriday.EXTENSION_AUTOLINK |
		blackfriday.EXTENSION_STRIKETHROUGH | blackfriday.EXTENSION_SPACE_HEADERS
)

var (
	renderer     = blackfriday.HtmlRenderer(flags, "", "")
	beginComment = []byte("{{/*")
	endComment   = []byte("*/}}")

	referencesRe  = regexp.MustCompile("(?m)$\\s*\\[\\w+\\]:.*?$")
	commandPrefix = "\uffffcommand"
	commandRe     = regexp.MustCompile(commandPrefix + "\\d+")
	fixupRe       = regexp.MustCompile("<p>(\\{\\{[^\\}]+\\})</p>")
)

func markdown(cur *bytes.Buffer, refs *bytes.Buffer) []byte {
	if cur.Len() > 0 {
		data := cur.Bytes()
		if refs.Len() > 0 {
			data = append(data, refs.Bytes()...)
		}
		md := blackfriday.Markdown(data, renderer, extensions)
		cur.Reset()
		return md
	}
	return nil
}

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
	var references bytes.Buffer
	for _, v := range referencesRe.FindAll(data, -1) {
		references.Write(v)
		references.WriteByte('\n')
	}
	var text bytes.Buffer
	var cmd bytes.Buffer
	var commands [][]byte
	command := false
	end := len(data)
	last := end - 1
	for ii := 0; ii < end; ii++ {
		c := data[ii]
		switch c {
		case '}':
			if command && ii < last && data[ii+1] == '}' {
				command = false
				cmd.WriteString("}}")
				if ii < last-1 && data[ii+2] == '\n' && false {
					// Line ends with command, creating
					// a md block
					out.Write(markdown(&text, &references))
					out.Write(cmd.Bytes())
					out.WriteByte('\n')
				} else {
					// Inline command, process it with
					// the rest of the markdown. To avoid
					// alterting special characters inside
					// a command, substitute it with a placeholder.
					fmt.Fprintf(&text, "%s%d", commandPrefix, len(commands))
					var c []byte
					c = append(c, cmd.Bytes()...)
					commands = append(commands, c)
				}
				cmd.Reset()
				ii++
			}
		case '{':
			if !command && ii < last && data[ii+1] == '{' {
				command = true
				ii++
				cmd.WriteString("{{")
			}
		default:
			if command {
				cmd.WriteByte(c)
			} else {
				text.WriteByte(c)
			}
		}
	}
	out.Write(markdown(&text, &references))
	md := commandRe.ReplaceAllFunc(out.Bytes(), func(b []byte) []byte {
		d := bytes.IndexByte(b, 'd')
		pos, err := strconv.Atoi(string(b[d+1:]))
		if err != nil {
			panic(err)
		}
		return commands[pos]
	})
	md = fixupRe.ReplaceAll(md, []byte("${1}"))
	return md, nil
}

func init() {
	template.RegisterConverter("md", toMarkdown)
}
