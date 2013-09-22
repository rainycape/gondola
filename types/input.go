package types

import (
	"gnd.la/i18n"
	"reflect"
	"regexp"
)

var (
	alphanumericRe = regexp.MustCompile("^[a-zA-Z0-9]+$")
)

// InputNamed takes a (generally) user-provided input string and parses it into the out
// parameter, which must be settable (this usually means you'll have to pass a pointer
// to this function). See the documentation on Parse() for a list of the supported types.
// If name is provided, it will be included in any error returned by this function.
// Additional constraints might be specified with the tag parameter, the ones currently
// supported are:
//  - required: Marks the field as required, will return an error if it's empty.
//  - optional: Marks the field as optional, will accept empty values.
//  - max_length: Sets the maximum length for the input.
//  - min_length: Sets the minimum length for the input.
//  - alphanumeric: Requires the input to be only letters and numbers
//
// Finally, the required parameter indicates if the value should be considered required
// or optional in absence of the "required" and "optional" tag fields.
func InputNamed(name string, input string, out interface{}, tag *Tag, required bool) error {
	v, err := SettableValue(out)
	if err != nil {
		return err
	}
	if err := parse(input, v); err != nil {
		return err
	}
	if v.Type().Kind() != reflect.Bool && tag != nil {
		if ((required && !tag.Optional()) || tag.Required()) && input == "" {
			if name != "" {
				return i18n.Errorf("%s is required", name)
			}
			return i18n.Errorf("required")
		}
		if maxlen, ok := tag.MaxLength(); ok && len(input) > maxlen {
			if name != "" {
				return i18n.Errorf("%s is too long (maximum length is %d)", name, maxlen)
			}
			return i18n.Errorf("too long (maximum length is %d)", maxlen)
		}
		if minlen, ok := tag.MinLength(); ok && len(input) < minlen {
			if name != "" {
				return i18n.Errorf("%s is too short (minimum length is %d)", name, minlen)
			}
			return i18n.Errorf("too short (minimum length is %d)", minlen)
		}
		if tag.Alphanumeric() && len(input) > 0 && !alphanumericRe.MatchString(input) {
			if name != "" {
				return i18n.Errorf("%s must be alphanumeric", name)
			}
			return i18n.Errorf("must be alphanumeric")
		}
	}
	return nil
}

// Input is a shorthand for InputNamed("", ...).
func Input(input string, out interface{}, tag *Tag, required bool) error {
	return InputNamed("", input, out, tag, required)
}
