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

func Set(w http.ResponseWriter, name string, value string) error {
	cookie := &http.Cookie{
		Name:  name,
		Value: value,
		Path:  "/",
	}
	http.SetCookie(w, cookie)
	return nil
}

func Get(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
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
	return Set(w, name, encoded)
}

func GetSecure(r *http.Request, name string) (interface{}, error) {
	cookieValue, err := Get(r, name)
	if err != nil {
		return nil, err
	}
	coder, err := getCookieCoder()
	if err != nil {
		return nil, err
	}
	var value interface{}
	err = coder.Decode(name, cookieValue, &value)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func Delete(w http.ResponseWriter, name string) {
	cookie := &http.Cookie{
		Name:   name,
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
}
