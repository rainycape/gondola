// Package admin provides functions for registering and
// executing administrative commands.
//
// To create a new administrative command, start by writing
// its function. Administrative functions take the same form
// as requests handlers e.g.:
//
//  func MyCommand(ctx *app.Context)
//
// Then register it using Register() or MustRegister().
//
//  func init() {
//	admin.MustRegister(MyCommand, nil)
//  }
//
// This will create a command named my-comand. To specify additional
// command options, like its name or the flags it might accept, set
// the Options parameter. See the documentation on Options on this
// page for information on each field or, alternatively, check the
// example down this page.
//
// If you're using gnd.la/app.App.ListenAndServe or gnd.la/app.App.MustListenAndServe,
// then you don't need to do anything else, since those functions will check if
// an administrative command was provided, perform it and exit.
// Alternatively, you can also call admin.Perform with a gnd.la/app.App
// instance manually if you're not using the functions previously mentioned
// or, if for some reason, you want to check for administrative commands
// before. e.g.
//
//  func main() {
//	// Set up ORM, config etc...
//	a := app.New()
//	// Set up context processors and finalizers, etc... on m
//	// Now check if there's an admin command and perform it
//	if !admin.Perform(a) {
//	    // No admin command supplied. Set up your handlers and
//	    // start listening.
//	    something := anExpensiveCalculationWhichTakesALotOfTime()
//	    a.Handle("^/hello/$", HelloHandler)
//	    a.MustListenAndServe(-1)
//	}
//	// Admin command was performed. Now just exit.
//  }
//
// Administrative commands might use the context methods ParamValue() to access flags
// values. Methods built on top of ParamValue(), like ParseParamValue(), are also
// supported.
// Any additional non-flag arguments are passed to the command handler and might be
// accessed using IndexValue() (0 represents the first non-flag argument). ParseIndexValue()
// and related methods are also supported.
//
//  admin.MustRegister(FlagsCommand, &admin.Options{
//	Help: "This command does nothing interesting",
//	Flags: admin.Flags(admin.IntFlag("foo", 0, "Help for foo flag"), admin.BoolFlag("bar", false, "Help for bar flag")),
//  })
//
//  func FlagCommand(ctx *app.Context) {
//	var foo int
//	var bar bool
//	ctx.ParseParamValue(&foo, "foo")
//	ctx.ParseParamValue(&bar, "bar")
//	// foo and bar now contain the parameters received in the command line
//  }
//
// Finally, to invoke and administrative command, pass it to your app binary e.g.
//
//  ./myapp my-command
//
// Keep in mind that any flags parsed by your application or the Gondola config package
// must come before the command name.
//
//  ./myapp -config=conf/production.conf my-command -mycommandflag=7
//
// To list all the available administrative commands together with their respective help, use
// the help command:
//
//  ./myapp help
//
// NOTE: These examples assume a UNIX environment. If you're using Windows type "myapp.exe" rather than "./myapp".
package admin
