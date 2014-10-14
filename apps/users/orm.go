package users

import (
	"gnd.la/orm"
	"gnd.la/orm/query"
)

// ByUsername returns a query.Q which finds a user given its
// username.
func ByUsername(username string) query.Q {
	return orm.Eq("User.NormalizedUsername", Normalize(username))
}

// ByEmail returns a query.Q which finds a user given its
// email.
func ByEmail(email string) query.Q {
	return orm.Eq("User.NormalizedEmail", Normalize(email))
}

// ById returns a query.Q which finds a user given its id.
func ById(id int64) query.Q {
	return orm.Eq("User.UserId", id)
}
