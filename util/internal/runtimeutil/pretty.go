// +build !windows,!appengine

package runtimeutil

import (
	"debug/gosym"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"sort"
	"strings"
)

type syms []*gosym.Sym

func (s syms) Len() int {
	return len(s)
}

func (s syms) Less(i, j int) bool {
	return s[i].Value < s[j].Value
}

func (s syms) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func makeTable(f debugFile) (*gosym.Table, error) {
	symdat, err := f.Section(".gosymtab").Data()
	if err != nil {
		return nil, err
	}
	pclndat, err := f.Section(".gopclntab").Data()
	if err != nil {
		return nil, err
	}
	pcln := gosym.NewLineTable(pclndat, f.Addr(".text"))
	tab, err := gosym.NewTable(symdat, pcln)
	if err != nil {
		return nil, err
	}

	return tab, nil

}

func funcName(s string) string {
	if pos := strings.LastIndex(s, "("); pos > 0 {
		return s[:pos]
	}
	return ""
}

func splitFunc(s string) (string, []string) {
	pos := strings.LastIndex(s, "(")
	if pos == -1 {
		return s, nil
	}
	f := s[:pos]
	args := s[pos+1 : len(s)-1]
	return f, strings.Split(args, ", ")
}

func prettyStack(lines []string, _html bool) (ret []string) {
	if len(lines) <= 1 {
		return lines
	}
	fp, err := os.Open(os.Args[0])
	if err != nil {
		return lines
	}
	defer fp.Close()
	f, err := openDebugFile(fp)
	if err != nil {
		return lines
	}
	defer f.Close()
	table, err := makeTable(f)
	if err != nil {
		return lines
	}
	// Grab first function name
	fname := funcName(lines[1])
	// Walk the callers until we find the
	// first function in the stack trace
	pcs := make([]uintptr, 1024)
	pcs = pcs[:runtime.Callers(2, pcs)]
	ii := 0
	for ; ii < len(pcs); ii++ {
		f := runtime.FuncForPC(pcs[ii])
		if f != nil && f.Name() == fname {
			break
		}
	}
	for jj := 1; ii < len(pcs) && jj < len(lines); ii++ {
		_, _, fn := table.PCToLine(uint64(pcs[ii]))
		if fn != nil && len(fn.Params) > 0 {
			params := make([]*gosym.Sym, len(fn.Params))
			// Symbols are in arbitraty order, their Value represents
			// their offset in the stack when the function is called,
			// so by ordering them according to their value we get the
			// same order that was specified in the source
			copy(params, fn.Params)
			sort.Sort(syms(params))
			fname, args := splitFunc(lines[jj])
			var reprs []string
			if strings.HasPrefix(fname, "runtime.") {
				reprs = args
			} else {
				pos := 0
			Params:
				for ii, v := range params {
					if pos >= len(args) {
						break
					}
					if strings.Contains(v.Name, ".~anon") {
						reprs = append(reprs, args[ii:]...)
						break
					}
					typ := reflectType(v.GoType)
					used := int(typ.Size() / reflect.TypeOf(uintptr(0)).Size())
					end := pos + used
					if end > len(args) {
						end = len(args)
					}
					values := args[pos:end]
					for _, val := range values {
						if val == "..." {
							reprs = append(reprs, values...)
							break Params
						}
					}
					pos = end
					repr, ok := fieldRepr(v, typ, values, _html)
					if ok {
						reprs = append(reprs, repr)
					} else {
						reprs = append(reprs, values...)
					}
				}
			}
			lines[jj] = fmt.Sprintf("%s(%s)", fname, strings.Join(reprs, ", "))
		}
		jj += 2
	}
	return lines
}
