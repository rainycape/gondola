// Package templateutil contains functions for parsing and walking go
// template trees.
package templateutil

import (
	"fmt"
	"regexp"
	"strings"
	"text/template/parse"
)

const (
	leftDelim  = "{{"
	rightDelim = "}}"
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
