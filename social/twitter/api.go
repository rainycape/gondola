package twitter

func (app *App) Get(token *Token, path string, data map[string]string, out interface{}) error {
	return sendReq(app, token, "GET", path, data, out)
}

func (app *App) Post(token *Token, path string, data map[string]string, out interface{}) error {
	return sendReq(app, token, "POST", path, data, out)
}

func (app *App) Verify(token *Token) (*User, error) {
	var user *User
	data := map[string]string{
		"skip_status": "1",
	}
	err := app.Get(token, verifyPath, data, &user)
	return user, err
}
