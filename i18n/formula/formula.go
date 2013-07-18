package formula

import (
	"fmt"
	"strconv"
	"strings"
)

// Formula is a function which accepts an int and returns
// the index of the plural form to use.
type Formula func(n int) (plural int)

// Make takes a Plural Form expression, like e.g. "nplurals=2; plural=n == 1 ? 0 : 1;"
// and returns its Formula function as well as the number of available plurals.
func Make(text string) (fn Formula, nplurals int, err error) {
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
	if nplurals == 1 {
		// Don't need to parse the formula
		fn = asianFormula
		return
	}
	form = strings.TrimSpace(form[sep+1:])
	if !strings.HasPrefix(form, "plural=") {
		err = fmt.Errorf("invalid plural formula %q, not starting with nplurals=", form)
		return
	}
	if form[len(form)-1] == ';' {
		form = form[:len(form)-1]
	}
	form = strings.TrimSpace(form[7:])
	if len(form) > 1 && form[0] == '(' && form[len(form)-1] == ')' {
		form = form[1 : len(form)-1]
	}
	switch strings.Replace(form, " ", "", -1) {
	case "n!=1":
		fn = romanicFormula
	case "n>1":
		fn = brazilianFrenchFormula
	case "n%10==1&&n%100!=11?0:n!=0?1:2":
		fn = latvianFormula
	case "n==1?0:n==2?1:2":
		fn = celticFormula
	case "n==1?0:(n==0||(n%100>0&&n%100<20))?1:2":
		fn = romanianFormula
	case "n%10==1&&n%100!=11?0:n%10>=2&&(n%100<10||n%100>=20)?1:2":
		fn = lithuanianFormula
	case "n%10==1&&n%100!=11?0:n%10>=2&&n%10<=4&&(n%100<10||n%100>=20)?1:2":
		fn = russianFormula
	case "(n==1)?0:(n>=2&&n<=4)?1:2":
		fn = czechFormula
	case "n==1?0:n%10>=2&&n%10<=4&&(n%100<10||n%100>=20)?1:2":
		fn = polishFormula
	case "n%100==1?0:n%100==2?1:n%100==3||n%100==4?2:3":
		fn = slovenianFormula
	}
	if fn == nil {
		fn, err = compileFormula(form)
	}
	return
}
