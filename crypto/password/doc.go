// Package password contains functions for securely storing
// and checking passwords.
//
// Passwords are encoded using a per-password salt and then
// hashed using PBKDF2 with the chosen algorithm (sha256 by default).
// Password provides the Check() method for verifying that
// the given plaintext matches the encoded password. This
// method is not vulnerable to timing attacks.
//
// Password objects can be stored directly by Gondola's ORM.
//
//  // "foo" is the username, "bar" is the password.
//  type User struct {
//	UserId int64 `orm:",primary_key,auto_increment"`
//	Username string
//	Password password.Password
//  }
//  // Creating a new user
//  user := &User{Username:"foo", Password: password.New("bar")}
//  // o is a gnd.la/orm.Orm object
//  o.MustSave(user)
//  // Signin in an existing user
//  var user *User
//  if err := o.One(orm.Eq("Username", "foo"), &user); err == nil {
//	if user.Password.Check("bar") == nil {
//	    // user has provided the correct password
//	}
//  }
//
// Password objects can also be stored on anything that accepts strings. See
// the examples to learn how to manually store and verify a password.
package password
