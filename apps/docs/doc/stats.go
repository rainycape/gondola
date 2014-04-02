package doc

import (
	"errors"
	"gnd.la/util/internal/astutil"
	"go/ast"
	"go/doc"
	"strings"
)

const (
	valueScore = 1
	funcScore  = 2
	typeScore  = 3
	fieldScore = 1
)

var (
	errInvalidPackage = errors.New("invalid package")
)

type Kind int

const (
	Const Kind = iota + 1
	Var
	Func
	Type
	Method
	Field
	IMethod
	Pkg
)

func (k Kind) DocScore() int {
	switch k {
	case Const, Var:
		return valueScore
	case Func, Method, IMethod:
		return funcScore
	case Type:
		return typeScore
	case Field:
		return fieldScore
	case Pkg:
		return noDocPenalty
	case 0:
		return 0
	}
	panic("unreachable")
}

type Undocumented struct {
	Kind Kind
	Name string
	Type string
	Node ast.Node
}

func (u *Undocumented) String() string {
	switch u.Kind {
	case Const:
		return "constant " + u.Name
	case Var:
		return "variable " + u.Name
	case Func:
		return "function " + u.Name
	case Type:
		return "type " + u.Name
	case Method:
		return "method (" + u.Type + ") " + u.Name
	case Field:
		return "field " + u.Name + " on type " + u.Type
	case IMethod:
		return "method " + u.Name + " on interface " + u.Type
	case Pkg:
		return "package documentation"
	}
	return "invalid Undocumented"
}

func (u *Undocumented) Id() string {
	switch u.Kind {
	case Const:
		return ConstId(u.Name)
	case Var:
		return VarId(u.Name)
	case Func:
		return FuncId(u.Name)
	case Type:
		return TypeId(u.Name)
	case Method:
		return MethodId(u.Type, u.Name)
	case Field:
		return TypeId(u.Type)
	case IMethod:
		return TypeId(u.Type)
	}
	return ""
}

const (
	noDocPenalty = 10
)

type Stats struct {
	p          *Package
	Documented int
	ToDocument int
	// Indicates if the package has documentation.
	HasDoc       bool
	Undocumented []*Undocumented
}

func (s *Stats) Package() *Package {
	return s.p
}

func (s *Stats) NoDocPenalty() int {
	if s.ToDocument == 0 {
		return 100
	}
	return noDocPenalty
}

func (s *Stats) DocPenalty() int {
	p := 0
	if !s.HasDoc {
		p += noDocPenalty
	}
	return p
}

func (s *Stats) docCoef() float64 {
	return float64(100 - s.DocPenalty())
}

func (s *Stats) DocumentedPercentage() float64 {
	if s.ToDocument == 0 {
		if s.HasDoc {
			return 100
		}
		return 0
	}
	if s.Documented == 0 && s.HasDoc {
		return float64(noDocPenalty)
	}
	return s.docCoef() * float64(s.Documented) / float64(s.ToDocument)
}

func (s *Stats) DocumentedIncrease(k Kind) float64 {
	if k == Pkg {
		return noDocPenalty
	}
	return s.docCoef() * float64(k.DocScore()) / float64(s.ToDocument)
}

func (s *Stats) valueStats(k Kind, values []*doc.Value, total *int, score *int) {
	for _, v := range values {
		if v.Doc != "" {
			// There's a comment just before the declaration.
			// Consider all the values documented
			c := len(v.Decl.Specs) * valueScore
			*score += c
			*total += c
		} else {
			// Check every value declared in this group
			for _, spec := range v.Decl.Specs {
				*total += valueScore
				sp := spec.(*ast.ValueSpec)
				if sp.Doc != nil || sp.Comment != nil {
					*score += valueScore
				} else {
					for _, n := range sp.Names {
						s.Undocumented = append(s.Undocumented, &Undocumented{
							Kind: k,
							Name: astutil.Ident(n),
							Node: spec,
						})
					}
				}
			}
		}
	}
}

func (s *Stats) funcStats(typ string, fns []*doc.Func, total *int, score *int) {
	for _, v := range fns {
		// Skip Error() and String() methods
		if typ != "" && (v.Name == "String" || v.Name == "Error") {
			continue
		}
		*total += funcScore
		if v.Doc != "" {
			*score += funcScore
		} else {
			und := &Undocumented{
				Kind: Func,
				Name: v.Name,
				Node: v.Decl,
			}
			if typ != "" {
				und.Type = typ
				und.Kind = Method
			}
			s.Undocumented = append(s.Undocumented, und)
		}
	}
}

func (s *Stats) typeStats(typs []*doc.Type, total *int, score *int) {
	for _, v := range typs {
		*total += typeScore
		if v.Doc != "" {
			*score += typeScore
		} else {
			s.Undocumented = append(s.Undocumented, &Undocumented{
				Kind: Type,
				Name: v.Name,
				Node: v.Decl,
			})
		}
		// Fields
		var k Kind
		ts := v.Decl.Specs[0].(*ast.TypeSpec)
		var fields []*ast.Field
		switch s := ts.Type.(type) {
		case *ast.StructType:
			fields = s.Fields.List
			k = Field
		case *ast.InterfaceType:
			fields = s.Methods.List
			k = IMethod
		}
		fs := k.DocScore()
		for _, f := range fields {
			*total += fs
			if f.Doc != nil || f.Comment != nil {
				*score += fs
			} else {
				var name string
				if len(f.Names) > 0 {
					name = astutil.Ident(f.Names[0])
				} else {
					// Embedded field
					name = astutil.Ident(f.Type)
					if name[0] == '*' {
						name = name[1:]
					}
					if dot := strings.IndexByte(name, '.'); dot >= 0 {
						name = name[dot+1:]
					}
				}
				s.Undocumented = append(s.Undocumented, &Undocumented{
					Kind: k,
					Name: name,
					Type: v.Name,
					Node: f,
				})
			}
		}
		s.valueStats(Const, v.Consts, total, score)
		s.valueStats(Var, v.Vars, total, score)
		s.funcStats("", v.Funcs, total, score)
		s.funcStats(v.Name, v.Methods, total, score)
	}
}

func NewStats(p *Package) (*Stats, error) {
	if p.dpkg == nil {
		return nil, errInvalidPackage
	}
	s := new(Stats)
	s.p = p
	total := 0
	score := 0
	if p.dpkg.Doc != "" {
		s.HasDoc = true
	} else {
		s.Undocumented = append(s.Undocumented, &Undocumented{
			Kind: Pkg,
		})
	}
	s.valueStats(Const, p.dpkg.Consts, &total, &score)
	s.valueStats(Var, p.dpkg.Vars, &total, &score)
	s.funcStats("", p.dpkg.Funcs, &total, &score)
	s.typeStats(p.dpkg.Types, &total, &score)
	s.Documented = score
	s.ToDocument = total
	return s, nil
}
