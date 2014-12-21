package fontawesome

import (
	htemplate "html/template"

	"gnd.la/template"
)

func fa(s string) htemplate.HTML {
	return htemplate.HTML("<i class=\"fa fa-" + s + "\"></i>")
}

func init() {
	template.AddFuncs(template.FuncMap{
		"fa": fa,
	})
}
