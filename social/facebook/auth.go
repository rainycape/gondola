package facebook

import (
	"fmt"
	"net/url"
	"strings"

	"gnd.la/net/oauth2"
)

func (app *App) Extend(token *oauth2.Token) (*oauth2.Token, error) {
	requestUrl := fmt.Sprintf("https://graph.facebook.com/oauth/access_token?client_id=%v&client_secret=%v&grant_type=fb_exchange_token&fb_exchange_token=%v",
		app.Id, app.Secret, token.Key)
	resp, err := app.client().HTTPClient.Get(requestUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	if responseHasError(resp) {
		return nil, decodeResponseError(resp)
	}
	newToken, err := oauth2.ParseToken(resp.Body)
	if err != nil {
		return nil, err
	}
	if newToken.Expires.IsZero() {
		// FB returned the same token because this token
		// was previously extended.
		newToken.Expires = token.Expires
	}
	return newToken, nil
}

func (app *App) AccountToken(token *oauth2.Token, accountId string) (*oauth2.Token, error) {
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
		return nil, fmt.Errorf("could not find token for account %s", accountId)
	}
	// The token expires at the same time as the main token
	return &oauth2.Token{Key: key, Expires: token.Expires}, nil
}

func Authorize(app *App, permissions []string) (*oauth2.Token, error) {
	// This URL is used by the FB JS SDK to get the token from the fragment
	redirect := "https://www.facebook.com/connect/login_success.html"
	auth := app.Authorization(redirect, permissions, "") + "&response_type=token"
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
	return oauth2.ParseToken(strings.NewReader(result.Fragment))
}
