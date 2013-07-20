package formula

func compileFormula(form string) (Formula, error) {
	// Try VM first
	fn, err := compileVmFormula(form)
	if err == nil {
		return fn, nil
	}
	return compileAstFormula(form)
}
