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
	walkNode(tree.Root, nil, f)
}

func walkNode(node, parent parse.Node, f func(n, p parse.Node)) {
	if node == nil {
		return
	}
	f(node, parent)
	switch x := node.(type) {
	case *parse.ActionNode:
		walkNode(x.Pipe, x, f)
	case *parse.PipeNode:
		for _, v := range x.Decl {
			walkNode(v, x, f)
		}
		for _, v := range x.Cmds {
			walkNode(v, x, f)
		}
	case *parse.CommandNode:
		for _, v := range x.Args {
			walkNode(v, x, f)
		}
	case *parse.ListNode:
		for _, v := range x.Nodes {
			walkNode(v, x, f)
		}
	case *parse.IfNode:
		if x.List != nil {
			walkNode(x.List, x, f)
		}
		if x.ElseList != nil {
			walkNode(x.ElseList, x, f)
		}
	case *parse.WithNode:
		if x.List != nil {
			walkNode(x.List, x, f)
		}
		if x.ElseList != nil {
			walkNode(x.ElseList, x, f)
		}
	case *parse.RangeNode:
		if x.List != nil {
			walkNode(x.List, x, f)
		}
		if x.ElseList != nil {
			walkNode(x.ElseList, x, f)
		}
	}
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
