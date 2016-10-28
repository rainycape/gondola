package input

import (
	"reflect"
	"regexp"

	"gnd.la/i18n"
	"gnd.la/util/structs"
	"gnd.la/util/types"
)

//go:generate gondola compile-messages

var (
	alphanumericRe = regexp.MustCompile("^[a-zA-Z0-9]+$")
)

// RequiredInputError returns an error indicating that the parameter
// with the given name is required but it's missing. Note that this function
// is only exported to avoid user-visible string duplication and users should
// not use it.
func RequiredInputError(name string) error {
	if name != "" {
		return i18n.Errorfc("form", "%s is required", name)
	}
	return i18n.Errorfc("form", "required")
}

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
func InputNamed(name string, input string, out interface{}, tag *structs.Tag, required bool) error {
	v, err := types.SettableValue(out)
	if err != nil {
		return err
	}
	if err := parse(input, v); err != nil {
		return err
	}
	if v.Type().Kind() != reflect.Bool && tag != nil {
		if ((required && !tag.Optional()) || tag.Required()) && input == "" {
			return RequiredInputError(name)
		}
		if maxlen, ok := tag.MaxLength(); ok && len(input) > maxlen {
			if name != "" {
				return i18n.Errorfc("form", "%s is too long (maximum length is %d)", name, maxlen)
			}
			return i18n.Errorfc("form", "too long (maximum length is %d)", maxlen)
		}
		if minlen, ok := tag.MinLength(); ok && len(input) < minlen {
			if name != "" {
				return i18n.Errorfc("form", "%s is too short (minimum length is %d)", name, minlen)
			}
			return i18n.Errorfc("form", "too short (minimum length is %d)", minlen)
		}
		if tag.Alphanumeric() && len(input) > 0 && !alphanumericRe.MatchString(input) {
			if name != "" {
				return i18n.Errorfc("form", "%s must be alphanumeric", name)
			}
			return i18n.Errorfc("form", "must be alphanumeric")
		}
	}
	return nil
}

// Input is a shorthand for InputNamed("", ...).
func Input(input string, out interface{}, tag *structs.Tag, required bool) error {
	return InputNamed("", input, out, tag, required)
}
