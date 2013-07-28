package formula

import (
	"fmt"
	"strconv"
	"strings"
)

// Formula is a function which accepts an int and returns
// the index of the plural form to use.
type Formula func(n int) (plural int)

// Extract takes a Plural Form expression e.g. "nplurals=2; plural=n == 1 ? 0 : 1;"
// and returns its formula (e.g. "n== 1 ? 0 : 1") as well as the number of plural
// forms (in the given example, 2). If the plural form can't be parsed, an error
// is returned.
func Extract(text string) (formula string, nplurals int, err error) {
	form := strings.TrimSpace(strings.ToLower(strings.Replace(text, "\\\n", "", -1)))
	if !strings.HasPrefix(form, "nplurals=") {
		err = fmt.Errorf("invalid Plural-Forms %q, not starting with nplurals=", text)
		return
	}
	form = form[9:]
	sep := strings.Index(form, ";")
	if sep == -1 {
		err = fmt.Errorf("invalid Plural-Forms %q, can't find number of plurals", text)
		return
	}
	nplurals, err = strconv.Atoi(form[:sep])
	if err != nil {
		err = fmt.Errorf("invalid Plural-Forms %q, error parsing nplurals: %s", text, err)
		return
	}
	form = strings.TrimSpace(form[sep+1:])
	if !strings.HasPrefix(form, "plural=") {
		err = fmt.Errorf("invalid plural formula %q, not starting with plural=", form)
		return
	}
	if form[len(form)-1] == ';' {
		form = form[:len(form)-1]
	}
	form = strings.TrimSpace(form[7:])
	if len(form) > 1 && form[0] == '(' && form[len(form)-1] == ')' {
		form = form[1 : len(form)-1]
	}
	formula = strings.TrimSpace(form)
	return
}

// Make takes a Plural Form expression, like e.g. "nplurals=2; plural=n == 1 ? 0 : 1;"
// and returns its Formula function as well as the number of available plurals.
func Make(text string) (fn Formula, nplurals int, err error) {
	var form string
	form, nplurals, err = Extract(text)
	if err != nil {
		return
	}
	var p Program
	p, err = Compile(form)
	if err != nil {
		return
	}
	fn = FromProgram(p)
	return
}

// Compile takes a plural formula (e.g. n == 1 ? 0 : 1) and returns
// the corresponding program, if possible. If the formula can't be
// parsed, an error is returned.
func Compile(formula string) (Program, error) {
	p, err := vmCompile([]byte(formula))
	if err != nil {
		return nil, err
	}
	return vmOptimize(p), nil
}

// FromProgram returns a Formula function from an already
// compiled Program.
func FromProgram(p Program) Formula {
	if fn := formulasTable[p.Id()]; fn != nil {
		return fn
	}
	fn, err := vmJit(p)
	if err == nil {
		return fn
	}
	return makeVmFunc(p)
}
