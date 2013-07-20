package formula

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"text/scanner"
)

type nodeType int

const (
	retNode nodeType = iota + 1
	ifNode
	binaryNode
	litNode
	nNode
)

type opType int

const (
	opEq opType = iota + 1
	opNeq
	opLt
	opLte
	opGt
	opGte
	opMod
	opAnd
	opOr
)

type node struct {
	Type nodeType
	Op   opType
	X    []*node
	Y    []*node
	Val  int
}

func compileAstFormula(form string) (Formula, error) {
	// Convert the formula into Go code
	var s scanner.Scanner
	var err error
	s.Init(strings.NewReader(form))
	s.Error = func(s *scanner.Scanner, msg string) {
		err = fmt.Errorf("error parsing plural formula %s: %s", s.Pos(), msg)
	}
	s.Mode = scanner.ScanIdents | scanner.ScanInts
	s.Whitespace = 0
	tok := s.Scan()
	var code []string
	var buf bytes.Buffer
	for tok != scanner.EOF && err == nil {
		switch tok {
		case scanner.Ident, scanner.Int:
			buf.WriteString(s.TokenText())
		case '?':
			code = append(code, fmt.Sprintf("if %s {\n", buf.String()))
			buf.Reset()
		case ':':
			code = append(code, fmt.Sprintf("return %s\n}\n", buf.String()))
			buf.Reset()
		default:
			buf.WriteRune(tok)
		}
		tok = s.Scan()
	}
	if err != nil {
		return nil, err
	}
	if len(code) == 0 && buf.Len() > 0 {
		code = append(code, fmt.Sprintf("if %s {\nreturn 1\n}\nreturn 0\n", buf.String()))
		buf.Reset()
	}
	if buf.Len() > 0 {
		code = append(code, fmt.Sprintf("\nreturn %s\n", buf.String()))
	}
	return compileGoFunc(strings.Join(code, ""))
}

func compileGoFunc(code string) (Formula, error) {
	expr, err := parser.ParseExpr(fmt.Sprintf("func (n int){\n%s\n}", code))
	if err != nil {
		return nil, err
	}
	// Convert go AST to a simplified AST
	fn := expr.(*ast.FuncLit)
	nodes := make([]*node, len(fn.Body.List))
	for ii, v := range fn.Body.List {
		nodes[ii], err = makeNode(v)
		if err != nil {
			return nil, err
		}
	}
	return makeAstFunc(nodes), nil
}

func intExpr(expr ast.Node) (int, error) {
	lit := expr.(*ast.BasicLit)
	return strconv.Atoi(lit.Value)
}

func makeNode(nod ast.Node) (*node, error) {
	switch n := nod.(type) {
	case *ast.ReturnStmt:
		v, err := intExpr(n.Results[0])
		if err != nil {
			return nil, err
		}
		return &node{Type: retNode, Val: v}, nil
	case *ast.IfStmt:
		x, err := makeNode(n.Cond)
		if err != nil {
			return nil, err
		}
		y, err := makeNode(n.Body.List[0])
		if err != nil {
			return nil, err
		}
		return &node{Type: ifNode, X: []*node{x}, Y: []*node{y}}, nil
	case *ast.BinaryExpr:
		x, err := makeNode(n.X)
		if err != nil {
			return nil, err
		}
		y, err := makeNode(n.Y)
		if err != nil {
			return nil, err
		}
		var op opType
		switch n.Op {
		case token.EQL:
			op = opEq
		case token.NEQ:
			op = opNeq
		case token.LSS:
			op = opLt
		case token.LEQ:
			op = opLte
		case token.GTR:
			op = opGt
		case token.GEQ:
			op = opGte
		case token.REM:
			op = opMod
		case token.LAND:
			op = opAnd
		case token.LOR:
			op = opOr
		default:
			return nil, fmt.Errorf("invalid operation %d", x.Op)
		}
		return &node{Type: binaryNode, Op: op, X: []*node{x}, Y: []*node{y}}, nil
	case *ast.ParenExpr:
		return makeNode(n.X)
	case *ast.Ident:
		if n.Name != "n" {
			return nil, fmt.Errorf("invalid ident in formula: %q", n.Name)
		}
		return &node{Type: nNode}, nil
	case *ast.BasicLit:
		v, err := intExpr(n)
		if err != nil {
			return nil, err
		}
		return &node{Type: litNode, Val: v}, nil
	}
	return nil, fmt.Errorf("can't handle node of type %T", nod)
}

func makeAstFunc(nodes []*node) Formula {
	return func(n int) int {
		return walkAst(nodes, n)
	}
}

func walkAst(nodes []*node, n int) int {
	for _, v := range nodes {
		switch v.Type {
		case retNode:
			return v.Val
		case ifNode:
			if walkAst(v.X, n) != 0 {
				return walkAst(v.Y, n)
			}
		case binaryNode:
			if v.Op == opAnd {
				if walkAst(v.X, n) != 0 && walkAst(v.Y, n) != 0 {
					return 1
				}
				return 0
			}
			if v.Op == opOr {
				if walkAst(v.X, n) != 0 || walkAst(v.Y, n) != 0 {
					return 1
				}
				return 0
			}
			x := walkAst(v.X, n)
			y := walkAst(v.Y, n)
			switch v.Op {
			case opEq:
				if x == y {
					return 1
				}
			case opNeq:
				if x != y {
					return 1
				}
			case opLt:
				if x < y {
					return 1
				}
			case opLte:
				if x <= y {
					return 1
				}
			case opGt:
				if x > y {
					return 1
				}
			case opGte:
				if x >= y {
					return 1
				}
			case opMod:
				return x % y
			case opOr, opAnd:
			default:
				panic("invalid op")

			}
			return 0
		case litNode:
			return v.Val
		case nNode:
			return n
		}
	}
	panic("unreachable")
}
