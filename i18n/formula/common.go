package formula

// Most common formulas implemented in Go code.
// If the formula isn't know, it will be interpreted.
// http://www.gnu.org/savannah-checkouts/gnu/gettext/manual/html_node/Plural-forms.html

func asianFormula(n int) int {
	return 0
}

func romanicFormula(n int) int {
	if n != 1 {
		return 1
	}
	return 0
}

func brazilianFrenchFormula(n int) int {
	if n > 1 {
		return 1
	}
	return 0
}

func latvianFormula(n int) int {
	if n%10 == 1 && n%100 != 11 {
		return 0
	}
	if n != 0 {
		return 1
	}
	return 2
}

func celticFormula(n int) int {
	if n == 1 {
		return 0
	}
	if n == 2 {
		return 1
	}
	return 2
}

func romanianFormula(n int) int {
	if n == 1 {
		return 0
	}
	if n == 0 || (n%100 > 0 && n%100 < 20) {
		return 1
	}
	return 2
}

func lithuanianFormula(n int) int {
	if n%10 == 1 && n%100 != 11 {
		return 0
	}
	if n%10 >= 2 && (n%100 < 10 || n%100 >= 20) {
		return 1
	}
	return 2
}

func russianFormula(n int) int {
	if n%10 == 1 && n%100 != 11 {
		return 0
	}
	if n%10 >= 2 && n%10 <= 4 && (n%100 < 10 || n%100 >= 20) {
		return 1
	}
	return 2
}

func czechFormula(n int) int {
	if n == 1 {
		return 0
	}
	if n >= 2 && n <= 4 {
		return 1
	}
	return 2
}

func polishFormula(n int) int {
	if n == 1 {
		return 0
	}
	if n%10 >= 2 && n%10 <= 4 && (n%100 < 10 || n%100 >= 20) {
		return 1
	}
	return 2
}

func slovenianFormula(n int) int {
	if n%100 == 1 {
		return 0
	}
	if n%100 == 2 {
		return 1
	}
	if n%100 == 3 || n%100 == 4 {
		return 2
	}
	return 3
}
