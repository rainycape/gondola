package formatutil

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"gnd.la/i18n"
	"gnd.la/util/stringutil"
	"gnd.la/util/types"
)

func Number(lang i18n.Languager, number interface{}) (string, error) {
	val := reflect.Indirect(reflect.ValueOf(number))
	if val.IsValid() {
		switch types.Kind(val.Kind()) {
		case types.Int:
			return formatNumber(lang, strconv.FormatInt(val.Int(), 10), ""), nil
		case types.Uint:
			return formatNumber(lang, strconv.FormatUint(val.Uint(), 10), ""), nil
		case types.Float:
			return formatStringNumber(lang, strconv.FormatFloat(val.Float(), 'f', -1, 64)), nil
		case types.String:
			return formatStringNumber(lang, val.String()), nil
		}
	}
	return "", fmt.Errorf("can't format type %T as number", number)
}

func formatStringNumber(lang i18n.Languager, s string) string {
	sep := strings.IndexByte(s, '.')
	if sep < 0 {
		return formatNumber(lang, s, "")
	}
	return formatNumber(lang, s[:sep], s[sep+1:])
}

func formatNumber(lang i18n.Languager, integer string, decimal string) string {
	/// THOUSANDS SEPARATOR
	tSep := i18n.Tc(lang, "formautil", ",")
	var buf bytes.Buffer
	ii := 0
	for _, c := range stringutil.Reverse(integer) {
		if ii == 3 {
			buf.WriteString(tSep)
			ii = 0
		}
		buf.WriteRune(c)
		ii++
	}
	s := stringutil.Reverse(buf.String())
	if decimal != "" {
		/// DECIMAL SEPARATOR
		dSep := i18n.Tc(lang, "formautil", ".")
		return s + dSep + decimal
	}
	return s
}
