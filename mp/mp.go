// Package mp stands for might panic and exports only the C function,
// which stands for "Check". It's inteded for solving the tedious
// error checking that might occur in Go codebases.
// Using this function you can save 2 lines most of the time you need
// to check for fatal error. Rather than writing:
//    if err != nil {
//        panic(err)
//    }
//
// You can now just write:
//
// mp.C(err)
//
package mp

// Function C panics iff err != nil. Otherwise it
// does nothing.
func C(err interface{}) {
	if err != nil {
		panic(err)
	}
}
