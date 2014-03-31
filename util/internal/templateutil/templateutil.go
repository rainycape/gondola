// Package templateutil contains functions for parsing and walking go
// template trees.
package templateutil

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template/parse"
)

const (
	leftDelim  = "{{"
	rightDelim = "}}"

	BeginTranslatableBlock = "begintrans"
	EndTranslatableBlock   = "endtrans"
)

var (
	funcNotDefinedRe = regexp.MustCompile("function \"(\\w+)\" not defined")
	varNotDefinedRe  = regexp.MustCompile("undefined variable \"\\$(\\w+)\"")
	defineRe         = regexp.MustCompile(`(\{\{\s*?define.*?\}\})`)
)

// WalkTree visits all the nodes in the tree, calling f
// for every node with its parent.
func WalkTree(tree *parse.Tree, f func(n, p parse.Node)) {
	WalkNode(tree.Root, nil, f)
}

// WalkNode visits all nodes starting from node, calling f
// with its parent. For the first call, the parent value is
// the one received as the second argument.
func WalkNode(node, parent parse.Node, f func(n, p parse.Node)) {
	if node == nil {
		return
	}
	f(node, parent)
	switch x := node.(type) {
	case *parse.ActionNode:
		WalkNode(x.Pipe, x, f)
	case *parse.PipeNode:
		for _, v := range x.Decl {
			WalkNode(v, x, f)
		}
		for _, v := range x.Cmds {
			WalkNode(v, x, f)
		}
	case *parse.CommandNode:
		for _, v := range x.Args {
			WalkNode(v, x, f)
		}
	case *parse.ListNode:
		for _, v := range x.Nodes {
			WalkNode(v, x, f)
		}
	case *parse.IfNode:
		if x.List != nil {
			WalkNode(x.List, x, f)
		}
		if x.ElseList != nil {
			WalkNode(x.ElseList, x, f)
		}
	case *parse.WithNode:
		if x.List != nil {
			WalkNode(x.List, x, f)
		}
		if x.ElseList != nil {
			WalkNode(x.ElseList, x, f)
		}
	case *parse.RangeNode:
		if x.List != nil {
			WalkNode(x.List, x, f)
		}
		if x.ElseList != nil {
			WalkNode(x.ElseList, x, f)
		}
	case *parse.TemplateNode:
		if x.Pipe != nil {
			WalkNode(x.Pipe, x, f)
		}
	}
}

// ReplaceNode replaces the n node in parent p with nn.
func ReplaceNode(n parse.Node, p parse.Node, nn parse.Node) error {
	found := false
	switch pn := p.(type) {
	/*case *parse.ActionNode:
	case *parse.PipeNode:*/
	case *parse.CommandNode:
		for ii, v := range pn.Args {
			if v == n {
				pn.Args[ii] = nn
				found = true
				break
			}
		}
	/*case *parse.ListNode:
	case *parse.IfNode:
	case *parse.WithNode:
	case *parse.RangeNode:*/
	default:
		return fmt.Errorf("can't replace node in %T", pn)
	}
	if !found {
		return fmt.Errorf("could not find node %v in parent %v", n, p)
	}
	return nil
}

// Parse parses the given text as a set of template trees, adding placeholders
// for undefined functions and variables.
func Parse(name string, text string) (map[string]*parse.Tree, error) {
	text = ReplaceVariableShorthands(text, '@', "")
	funcs := make(map[string]interface{})
	var treeSet map[string]*parse.Tree
	var err error
	for {
		treeSet, err = parse.Parse(name, text, leftDelim, rightDelim, funcs)
		if err != nil {
			if m := funcNotDefinedRe.FindStringSubmatch(err.Error()); m != nil {
				funcs[m[1]] = func() {}
				continue
			}
			if m := varNotDefinedRe.FindStringSubmatch(err.Error()); m != nil {
				prepend := fmt.Sprintf("{{ $%s := . }}\n", m[1])
				text = prepend + defineRe.ReplaceAllString(text, "$0"+strings.Replace(prepend, "$", "$$", -1))
				continue
			}
			return nil, err
		}
		break
	}
	return treeSet, nil
}

func ReplaceVariableShorthands(text string, chr byte, name string) string {
	repl := fmt.Sprintf("$%s.", name)
	var buf bytes.Buffer
	buf.Grow(len(text))
	cmd := false
	quoted := false
	for ii := range text {
		v := text[ii]
		if v == chr && cmd && !quoted {
			buf.WriteString(repl)
			continue
		}
		if cmd && v == '}' {
			cmd = ii < len(text)-1 && text[ii+1] != '}'
		} else if !cmd && v == '{' {
			cmd = ii < len(text)-1 && text[ii+1] == '{'
		}
		if cmd {
			if v == '"' && (ii == 0 || text[ii-1] != '\\') {
				quoted = !quoted
			}
		}
		buf.WriteByte(v)
	}
	return buf.String()
}

// ReplaceTranslatableBlocks replaces begintrans/endtrans blocks
// with an equivalent action using the translation function named
// by fn.
func ReplaceTranslatableBlocks(tr *parse.Tree, fn string) error {
	var err error
	begin := fmt.Sprintf("{{%s}}", BeginTranslatableBlock)
	end := fmt.Sprintf("{{%s}}", EndTranslatableBlock)
	WalkTree(tr, func(n, p parse.Node) {
		if err != nil {
			return
		}
		if n.Type() == parse.NodeAction && n.String() == begin {
			list, ok := p.(*parse.ListNode)
			if !ok {
				loc, ctx := tr.ErrorContext(n)
				err = fmt.Errorf("%s:%s:%s not in ListNode (%T)", loc, ctx, BeginTranslatableBlock, p)
				return
			}
			cmd := &parse.CommandNode{
				NodeType: parse.NodeCommand,
				Pos:      n.Position(),
			}
			pipe := &parse.PipeNode{
				NodeType: parse.NodePipe,
				Pos:      n.Position(),
				Cmds:     []*parse.CommandNode{cmd},
			}
			repl := &parse.ActionNode{
				NodeType: parse.NodeAction,
				Pos:      n.Position(),
				Pipe:     pipe,
			}
			if repl != nil {
			}
			pos := -1
			for ii, v := range list.Nodes {
				if v == n {
					pos = ii
					break
				}
			}
			var pipes []parse.Node
			var buf bytes.Buffer
			endPos := -1
		Nodes:
			for ii, v := range list.Nodes[pos+1:] {
				switch x := v.(type) {
				case *parse.TextNode:
					buf.Write(x.Text)
				case *parse.ActionNode:
					if x.String() == end {
						endPos = ii
						break Nodes
					}
					if len(x.Pipe.Decl) > 0 {
						loc, ctx := tr.ErrorContext(n)
						err = fmt.Errorf("%s:%s:%s translatable block can't contain a pipe with declaractions", loc, ctx, v)
					}
					buf.WriteString("%v")
					pipes = append(pipes, x.Pipe)
				default:
					loc, ctx := tr.ErrorContext(n)
					err = fmt.Errorf("%s:%s:%s translatable block can't contain %T", loc, ctx, v)
					return
				}
			}
			if buf.Len() > 0 {
				text := buf.String()
				quoted := strconv.Quote(text)
				innerPipe := &parse.PipeNode{
					NodeType: parse.NodePipe,
					Pos:      n.Position(),
					Cmds: []*parse.CommandNode{
						&parse.CommandNode{
							NodeType: parse.NodeCommand,
							Pos:      n.Position(),
							Args: []parse.Node{parse.NewIdentifier(fn), &parse.StringNode{
								NodeType: parse.NodeString,
								Pos:      n.Position(),
								Quoted:   quoted,
								Text:     text,
							}},
						},
					},
				}
				cmd.Args = append(cmd.Args, parse.NewIdentifier("printf"), innerPipe)
				cmd.Args = append(cmd.Args, pipes...)
			}
			nodes := list.Nodes[:pos]
			nodes = append(nodes, repl)
			nodes = append(nodes, list.Nodes[endPos+pos+1:]...)
			list.Nodes = nodes
		}
	})
	return err
}
