package password_test

import (
	"fmt"
	"gnd.la/users/password"
)

func ExampleNew() {
	// Provided by the user, usually at registration.
	plain := "alberto"
	// p contains the encoded password, which can
	// be stored in the database.
	p := password.New(plain)
	// This prints the encoded password.
	fmt.Println(p)
	// This will print the same as the previous line
	// but its type will be string. It might be useful
	// for some storage drivers that expect values of
	// type string.
	fmt.Println(p.String())
}

func ExamplePassword_Check() {
	// This will usually come from the database. In this case, the encoded
	// password is "gondola".
	encoded := "sha1:4096:JJf2f46fmbw06LwXJ308:9b4d23006b93e1d6bb052c1545d9532d1433736b"
	// This will usually come from a form, filled in by the user for signing in.
	plain := "gondola"
	p := password.Password(encoded)
	if p.Check(plain) == nil {
		// Plaintext password matches the stored one.
		fmt.Println("Password is", plain)
	}
	// Output: Password is gondola
}
