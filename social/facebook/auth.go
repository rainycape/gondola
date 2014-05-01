package facebook

import (
	"fmt"
	"net/url"
	"strings"
)

func (app *App) AuthURL(redirectUri string, permissions []string, state string) string {
	scope := strings.Join(permissions, ",")
	facebookUrl := fmt.Sprintf("https://www.facebook.com/dialog/oauth?client_id=%v&redirect_uri=%v&scope=%v&state=%v",
		app.Id, url.QueryEscape(redirectUri), scope, state)
	return facebookUrl
}

func (app *App) ExchangeCode(code string, redirectUri string, extend bool) (*Token, error) {
	exchangeUrl := fmt.Sprintf("https://graph.facebook.com/oauth/access_token?client_id=%v&redirect_uri=%v&client_secret=%v&code=%v",
		app.Id, url.QueryEscape(redirectUri), app.Secret, code)
	resp, err := app.client().Get(exchangeUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	if responseHasError(resp) {
		return nil, decodeResponseError(resp)
	}
	b, err := resp.ReadAll()
	if err != nil {
		return nil, err
	}
	token, err := ParseToken(string(b))
	if err == nil && extend {
		token, err = app.ExtendToken(token)
	}
	return token, err
}

func (app *App) ExtendToken(token *Token) (*Token, error) {
	requestUrl := fmt.Sprintf("https://graph.facebook.com/oauth/access_token?client_id=%v&client_secret=%v&grant_type=fb_exchange_token&fb_exchange_token=%v",
		app.Id, app.Secret, token.Key)
	resp, err := app.client().Get(requestUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	if responseHasError(resp) {
		return nil, decodeResponseError(resp)
	}
	b, err := resp.ReadAll()
	if err != nil {
		return nil, err
	}
	newToken, err := ParseToken(string(b))
	if err == ErrMissingExpires {
		/* FB returned the same token because this token
		was previously extended */
		newToken, err = token, nil
	}
	return newToken, err
}

func (app *App) AccountToken(token *Token, accountId string) (*Token, error) {
	resp, err := app.Get("/me/accounts", nil, token.Key)
	if err != nil {
		return nil, err
	}
	data := resp["data"].([]interface{})
	key := ""
	for _, v := range data {
		account := v.(map[string]interface{})
		id := account["id"].(string)
		if id == accountId {
			key = account["access_token"].(string)
			break
		}
	}
	if key == "" {
		return nil, fmt.Errorf("Could not find token for account %s", accountId)
	}
	/* The token expires at the same time as the main token */
	return &Token{key, token.Expires}, nil
}

func Authorize(app *App, permissions []string) (*Token, error) {
	// This URL is used by the FB JS SDK to get the token from the fragment
	redirect := "https://www.facebook.com/connect/login_success.html"
	auth := app.AuthURL(redirect, permissions, "") + "&response_type=token"
	fmt.Printf("Please, open the following URL in your browser:\n%s\n", auth)
	fmt.Printf("Then, paste the resulting URL after authorizing the app\nResulting URL: ")
	var input string
	_, err := fmt.Scanf("%s", &input)
	if err != nil {
		return nil, err
	}
	result, err := url.Parse(input)
	if err != nil {
		return nil, err
	}
	return ParseToken(result.Fragment)
}
