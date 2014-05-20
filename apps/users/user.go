package users

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gnd.la/app"
	"gnd.la/crypto/password"
	"gnd.la/i18n"
	"gnd.la/net/mail"
	"gnd.la/orm"
	"gnd.la/signal"
	"gnd.la/util/structs"
)

var (
	userType  reflect.Type = nil
	innerType              = reflect.TypeOf(User{})
	fbType                 = reflect.TypeOf(&Facebook{})
	twType                 = reflect.TypeOf(&Twitter{})
	gogType                = reflect.TypeOf(&Google{})
)

type missingFieldError struct {
	typ      reflect.Type
	name     string
	fieldTyp reflect.Type
}

func (e *missingFieldError) Error() string {
	typ := e.typ
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return fmt.Sprintf("user type %s requires a field named %q of type %s e.g. type %s struct {\n\t...\n\t%s %s\n}",
		typ.Name(), e.name, e.fieldTyp, typ.Name(), e.name, e.fieldTyp)
}

func SetType(val interface{}) {
	var typ reflect.Type
	if tt, ok := val.(reflect.Type); ok {
		typ = tt
	} else {
		typ = reflect.TypeOf(val)
	}
	if typ != nil {
		for typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
	}
	checkUserType(typ)
	userType = typ
}

func checkUserType(typ reflect.Type) {
	if typ == nil {
		panic(fmt.Errorf("User type is not set - configure it with users.SetType(&MyUserType{})"))
	}
	s, err := structs.NewStruct(typ, nil)
	if err != nil {
		panic(err)
	}
	if !s.Embeds(innerType) {
		panic(fmt.Errorf("invalid User type %s: must embed %s e.g type %s struct {\t\t%s\n\t...\n}", typ, innerType, typ.Name(), innerType))
	}
	if FacebookApp != nil && !s.Has("Facebook", fbType) {
		panic(&missingFieldError{typ, "Facebook", fbType})
	}
	if TwitterApp != nil && !s.Has("Twitter", twType) {
		panic(&missingFieldError{typ, "Twitter", twType})
	}
	if GoogleApp != nil && !s.Has("Google", gogType) {
		panic(&missingFieldError{typ, "Google", gogType})
	}
}

func Normalize(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

type User struct {
	UserId             int64             `form:"-" sql:"id,primary_key,auto_increment" json:"id"`
	Username           string            `form:",max_length=16,min_length=4,alphanumeric,label=Username" json:"username"`
	NormalizedUsername string            `form:"-" sql:",unique" json:"-"`
	Email              string            `form:",max_length=50,label=Email" json:"-"`
	NormalizedEmail    string            `form:"-" sql:",unique" json:"-"`
	Password           password.Password `form:"-,min_length=6,label=Password" json:"-"`
	Created            time.Time         `json:"-" form:"-"`
	AutomaticUsername  bool              `form:"-" json:"-"`
	Admin              bool              `form:"-" orm:",default=false" json:"admin"`
	Image              string            `form:"-" sql:",omitempty,nullempty" json:"-"`
	ImageFormat        string            `form:"-" sql:",omitempty,nullempty" json:"-"`
}

func (u *User) Id() int64 {
	return u.UserId
}

func (u *User) IsAdmin() bool {
	return u.Admin
}

func (u *User) Save() {
	u.NormalizedUsername = Normalize(u.Username)
	u.NormalizedEmail = Normalize(u.Email)
}

func (u *User) ValidateUsername(ctx *app.Context) error {
	norm := Normalize(u.Username)
	_, userVal := newEmptyUser()
	found, err := ctx.Orm().One(orm.Eq("User.NormalizedUsername", norm), userVal)
	if err != nil {
		panic(err)
	}
	if found {
		return i18n.Errorf("username %q is already in use", u.Username)
	}
	return nil
}

func (u *User) ValidateEmail(ctx *app.Context) error {
	addr, err := mail.Validate(u.Email, true)
	if err != nil {
		return i18n.Errorf("this does not look like a valid email address")
	}
	u.Email = addr
	norm := Normalize(u.Email)
	_, userVal := newEmptyUser()
	found, err := ctx.Orm().One(orm.Eq("User.NormalizedEmail", norm), userVal)
	if err != nil {
		panic(err)
	}
	if found {
		return i18n.Errorf("email %q is already in use", u.Email)
	}
	return nil
}

func setUserValue(v reflect.Value, key string, value interface{}) {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	v.FieldByName(key).Set(reflect.ValueOf(value))
}

func getUserValue(v reflect.Value, key string) interface{} {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v.FieldByName(key).Interface()
}

func asGondolaUser(v reflect.Value) app.User {
	return v.Interface().(app.User)
}

func JSONEncode(user interface{}) ([]byte, error) {
	v := reflect.ValueOf(user)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	inner := getUserValue(v, "User").(User)
	val := map[string]interface{}{
		"id":       inner.UserId,
		"username": inner.Username,
		"admin":    inner.Admin,
	}
	if img, _ := Image(v.Interface()); img != "" {
		val["image"] = img
	}
	if fb, ok := getUserValue(v, "Facebook").(*Facebook); ok {
		val["facebook"] = fb
	}
	if tw, ok := getUserValue(v, "Twitter").(*Twitter); ok {
		val["twitter"] = tw
	}
	if gog, ok := getUserValue(v, "Google").(*Google); ok {
		val["google"] = gog
	}
	b, err := json.Marshal(val)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func writeJSONEncoded(ctx *app.Context, user reflect.Value) {
	json, err := JSONEncode(user.Interface())
	if err != nil {
		panic(err)
	}
	ctx.SetHeader("Content-Type", "application/json")
	ctx.SetHeader("Content-Length", strconv.Itoa(len(json)))
	if _, err := ctx.Write(json); err != nil {
		panic(err)
	}
}

func newUser(username string) reflect.Value {
	u := reflect.New(userType)
	setUserValue(u, "Username", username)
	setUserValue(u, "Created", time.Now().UTC())
	return u
}

func newEmptyUser() (reflect.Value, interface{}) {
	user := reflect.New(userType)
	return user, user.Interface()
}

func init() {
	signal.Listen(app.DID_PREPARE, func() {
		checkUserType(userType)
	})
}
