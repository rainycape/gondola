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
)

var (
	innerType = reflect.TypeOf(User{})
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

func Normalize(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

type User struct {
	UserId             int64             `form:"-" orm:"id,primary_key,auto_increment" json:"id"`
	Username           string            `form:",max_length=16,min_length=4,alphanumeric,label=Username" json:"username"`
	NormalizedUsername string            `form:"-" orm:",unique" json:"-"`
	Email              string            `form:",max_length=50,label=Email" json:"-"`
	NormalizedEmail    string            `form:"-" orm:",unique" json:"-"`
	Password           password.Password `form:"-,min_length=6,label=Password" json:"-"`
	Created            time.Time         `json:"-" form:"-"`
	AutomaticUsername  bool              `form:"-" json:"-"`
	Admin              bool              `form:"-" orm:",default=false" json:"admin"`
	Image              string            `form:"-" orm:",omitempty,nullempty" json:"-"`
	ImageFormat        string            `form:"-" orm:",omitempty,nullempty" json:"-"`
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
	return validateNewUsername(ctx, u.Username)
}

func (u *User) ValidateEmail(ctx *app.Context) error {
	addr, err := validateNewEmail(ctx, u.Email)
	if err != nil {
		return err
	}
	u.Email = addr
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
	val := v.FieldByName(key)
	if val.IsValid() {
		return val.Interface()
	}
	return nil
}

func asGondolaUser(v reflect.Value) app.User {
	return v.Interface().(app.User)
}

func JSONEncode(ctx *app.Context, user interface{}) ([]byte, error) {
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
	if img, _ := Image(ctx, v.Interface()); img != "" {
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
	if gh, ok := getUserValue(v, "Github").(*Github); ok {
		val["github"] = gh
	}
	b, err := json.Marshal(val)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func writeJSONEncoded(ctx *app.Context, user reflect.Value) {
	json, err := JSONEncode(ctx, user.Interface())
	if err != nil {
		panic(err)
	}
	ctx.SetHeader("Content-Type", "application/json")
	ctx.SetHeader("Content-Length", strconv.Itoa(len(json)))
	if _, err := ctx.Write(json); err != nil {
		panic(err)
	}
}

func newUser(ctx *app.Context, username string) reflect.Value {
	u := reflect.New(getUserType(ctx))
	setUserValue(u, "Username", username)
	setUserValue(u, "Created", time.Now().UTC())
	return u
}

func newEmptyUser(ctx *app.Context) (reflect.Value, interface{}) {
	user := reflect.New(getUserType(ctx))
	return user, user.Interface()
}

func getUserType(ctx *app.Context) reflect.Type {
	return data(ctx).userType
}
