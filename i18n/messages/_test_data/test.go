package main

import (
	"fmt"
	"gondola/i18n"
	"gondola/mux"
)

type Test struct {
	/// Translatable string via field tag
	// This is not part of the translation comment.
	Username string `forms:",label:Username"`
	/// This field should get its help extracted
	Id int `config:",help:This is the user id" forms:",label:Identifier"`
}

const (
	/// This is a constant translatable string
	_ i18n.String = "Testing constant string"
	/// This constant string is created by concatenating two strings.
	_ i18n.String = "Concatenated" + " constant string"
)

/// This is a var string declared via type
var foo i18n.String = "Testing var string"

/// This is a var string declared via cast
var bar = i18n.String("Testing var casted string")

func main() {
	/// Please, translate this.
	/// This comment is still part of the translation comment.
	// This comment is not part of the translation comment.
	fmt.Println(i18n.T("Hello world\n", nil))

	/// This translation has a context
	fmt.Println(i18n.Tc("second", "Hello world\n", nil))

	/// This is a long translation, to test line splitting in quoted strings.
	fmt.Println(i18n.T("Lorem ipsum dolor sit amet, consectetur adipiscing elit. In sed ante ut massa ultrices auctor. Vivamus rutrum ut ante et aliquet. Proin ut rutrum enim, a elementum ligula. Morbi malesuada.", nil))

	// This is not part of the translation comment.
	//
	// T: Translate this too.
	fmt.Println(i18n.T("Bye"+" "+"wor\"ld", nil))

	/*/ This whole comment is part of the translation
	  comment.
	   And it keeps newlines, but strips leading whitespace.
	*/
	// This, however, is not.
	fmt.Println(i18n.T("Hello again "+"world", nil))

	for ii := 0; ii < 5; ii++ {
		/// Using i18n.Sprintfn. The format string is fixed by i18n.Sprintfn
		/// so it doesn't show any extra arguments.
		fmt.Println(i18n.Sprintfn("Hello one world", "Hello %d worlds", ii, nil, ii))
	}
}

func testing(ctx *mux.Context) {
	/// This is a very long comment to test the 80 columns per line splitting used automatically by gondola/i18n. Isn't it cool?
	ctx.T("Testing more translations")
}

func testing2(ctx mux.Context) {
	ctx.T("Testing even more translations")
	var t1 i18n.String = "Var inside function"
	/// This is a var string declared via cast
	t2 := i18n.String("Testing var string")
	fmt.Println(t1, t2)
}
