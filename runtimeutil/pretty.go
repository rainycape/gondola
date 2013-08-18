package runtimeutil

import (
	"debug/gosym"
	"fmt"
	"os"
	"runtime"
	"strings"
)

type section interface {
	Data() ([]byte, error)
}

type debugFile interface {
	Section(name string) section
	Addr(name string) uint64
	Close() error
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

func prettyStack(lines []string, _html bool) []string {
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
			// Symbols are in reverse order
			for ii, p := range fn.Params {
				params[len(fn.Params)-1-ii] = p
			}
			fname, args := splitFunc(lines[jj])
			kk := 0
			for _, v := range params {
				if kk >= len(args) || args[kk] == "..." || strings.Contains(v.Name, ".~anon") {
					break
				}
				repr, used, ok := fieldRepr(table, fn, v, args[kk:])
				if ok {
					args[kk] = repr
				}
				if used > 1 {
					// Remove consumed args
					end := kk + used
					if end > len(args) {
						end = len(args)
					}
					args = append(args[:kk+1], args[end:]...)
				}
				kk++
			}
			lines[jj] = fmt.Sprintf("%s(%s)", fname, strings.Join(args, ", "))
		}
		jj += 2
	}
	return lines
}
