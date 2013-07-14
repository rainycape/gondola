package messages

import (
	"go/ast"
	"go/token"
	"strings"
)

func comments(fset *token.FileSet, f *ast.File, pos *token.Position) string {
	for _, v := range f.Comments {
		end := fset.Position(v.End())
		if end.Filename == pos.Filename && end.Line == pos.Line-1 {
			var lines []string
			for _, c := range v.List {
				text := c.Text
				if strings.HasPrefix(text, "//") {
					text = text[2:]
				} else if strings.HasPrefix(text, "/*") {
					text = text[2 : len(text)-4]
				}
				text = strings.TrimSpace(text)
				if text == "" || text == "/" || text[0] != '/' {
					continue
				}
				text = strings.TrimSpace(text[1:])
				lines = append(lines, text)
			}
			if len(lines) > 0 {
				return strings.Join(lines, "\n")
			}
		}
	}
	return ""
}
