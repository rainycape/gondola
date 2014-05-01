package twitter

import (
	"fmt"
	"sync"

	"gnd.la/net/oauth"
)

var pendingTokens struct {
	sync.Mutex
	secrets map[string]string
}

func authorize(app *App, url string, callback string) (string, error) {
	c := newConsumer(app)
	c.AuthorizationURL = url
	c.CallbackURL = callback
	s, rt, err := c.Authorization()
	if err != nil {
		return "", err
	}
	pendingTokens.Lock()
	defer pendingTokens.Unlock()
	if pendingTokens.secrets == nil {
		pendingTokens.secrets = make(map[string]string)
	}
	pendingTokens.secrets[rt.Key] = rt.Secret
	return s, nil
}

func purgeToken(token string) {
	pendingTokens.Lock()
	defer pendingTokens.Unlock()
	delete(pendingTokens.secrets, token)
}

func (app *App) Authorize(callback string) (string, error) {
	return authorize(app, AUTHORIZATION_URL, callback)
}

func (app *App) Authenticate(callback string) (string, error) {
	return authorize(app, AUTHENTICATION_URL, callback)
}

func (app *App) Exchange(token string, verifier string) (*Token, error) {
	pendingTokens.Lock()
	secret := pendingTokens.secrets[token]
	delete(pendingTokens.secrets, token)
	pendingTokens.Unlock()
	if secret == "" {
		return nil, fmt.Errorf("can't find secret for token %s", token)
	}
	c := newConsumer(app)
	tk, err := c.Exchange(&oauth.Token{Key: token, Secret: secret}, verifier)
	if err != nil {
		return nil, err
	}
	return &Token{
		Key:    tk.Key,
		Secret: tk.Secret,
	}, nil
}
