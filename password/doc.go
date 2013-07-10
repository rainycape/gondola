// Package password contains functions for securely storing
// and checking passwords.
//
// Passwords are encoded using a per-password salt and then
// hashed with the chosen algorithm (sha256 by default).
// Password provides the Check() method for verifying that
// the given plaintext matches the encoded password. This
// method is not vulnerable to timing attacks.
package password
