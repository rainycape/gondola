package fontawesome

import (
	htemplate "html/template"

	"gnd.la/template"
)

func fa(s string) htemplate.HTML {
	return htemplate.HTML("<i class=\"fa fa-" + s + "\"></i>")
}

func init() {
	template.AddFunc(&template.Func{Name: "fa", Fn: fa})
}
