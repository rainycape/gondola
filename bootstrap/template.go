package bootstrap

import (
	"gnd.la/template"
	htemplate "html/template"
)

func fa(s string) htemplate.HTML {
	return htemplate.HTML("<i class=\"fa fa-" + s + "\"></i>")
}

func fa3(s string) htemplate.HTML {
	return htemplate.HTML("<i class=\"icon-" + s + "\"></i>")
}

func init() {
	template.AddFuncs(template.FuncMap{
		"fa":  fa,
		"fa3": fa3,
	})
}
