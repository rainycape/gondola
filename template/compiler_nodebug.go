// +build !template_compiler_debug

package template

func (p *program) debugDump()                           {}
func compilerDebugf(format string, args ...interface{}) {}
func compilerDebugln(args ...interface{})               {}
