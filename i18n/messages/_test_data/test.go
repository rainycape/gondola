package main

import (
	"fmt"
	"gnd.la/app"
	"gnd.la/i18n"
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
	fmt.Println(i18n.T(nil, "Hello world\n"))

	/// This translation has a context
	fmt.Println(i18n.Tc(nil, "second", "Hello world\n"))

	/// This is a long translation, to test line splitting in quoted strings.
	fmt.Println(i18n.T(nil, "Lorem ipsum dolor sit amet, consectetur adipiscing elit. In sed ante ut massa ultrices auctor. Vivamus rutrum ut ante et aliquet. Proin ut rutrum enim, a elementum ligula. Morbi malesuada."))

	// This is not part of the translation comment.
	//
	// T: Translate this too.
	fmt.Println(i18n.T(nil, "Bye"+" "+"wor\"ld"))

	/*/ This whole comment is part of the translation
	  comment.
	   And it keeps newlines, but strips leading whitespace.
	*/
	// This, however, is not.
	fmt.Println(i18n.T(nil, "Hello again "+"world"))

	for ii := 0; ii < 5; ii++ {
		/// Using i18n.Sprintfn. The format string is fixed by i18n.Sprintfn
		/// so it doesn't show any extra arguments.
		fmt.Println(i18n.Sprintfn(nil, "Hello one world", "Hello %d worlds", ii, ii))
	}
}

func testing(ctx *app.Context) {
	/// This is a very long comment to test the 80 columns per line splitting used automatically by gnd.la/i18n. Isn't it cool?
	ctx.T("Testing more translations")
}

func testing2(ctx app.Context) {
	ctx.T("Testing even more translations")
	var t1 i18n.String = "Var inside function"
	/// This is a var string declared via cast
	t2 := i18n.String("Testing var string")
	fmt.Println(t1, t2)
}
