package cookie

import (
	"code.google.com/p/gorilla/securecookie"
	"errors"
	"net/http"
)

var (
	ErrNoHashKey = errors.New("No cookie hash key specified")
	HashKey      = ""
	EncryptKey   = ""
)

func getCookieCoder() (*securecookie.SecureCookie, error) {
	if HashKey == "" {
		return nil, ErrNoHashKey
	}
	var encryptKey []byte
	if EncryptKey != "" {
		encryptKey = []byte(EncryptKey)
	}
	return securecookie.New([]byte(HashKey), encryptKey), nil
}

func SetSecure(w http.ResponseWriter, name string, value interface{}) error {
	coder, err := getCookieCoder()
	if err != nil {
		return err
	}
	encoded, err := coder.Encode(name, value)
	if err != nil {
		return err
	}
	cookie := &http.Cookie{
		Name:  name,
		Value: encoded,
		Path:  "/",
	}
	http.SetCookie(w, cookie)
	return nil
}

func GetSecure(r *http.Request, name string) (interface{}, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return nil, err
	}
	coder, err := getCookieCoder()
	if err != nil {
		return nil, err
	}
	var value interface{}
	err = coder.Decode(name, cookie.Value, &value)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func Delete(w http.ResponseWriter, name string) {
	cookie := &http.Cookie{
		Name:   name,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
}
