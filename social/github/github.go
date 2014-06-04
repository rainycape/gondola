package github

import (
	"errors"
	"fmt"

	"gnd.la/net/httpclient"
)

func (a *App) User(name string, accessToken string) (*User, error) {
	var user *User
	var p string
	if name == "" {
		// Authenticated user
		p = "/user"
	} else {
		// Given user
		p = "/users/" + name
	}
	err := a.Get(p, nil, accessToken, &user)
	return user, err
}

func (a *App) Emails(accessToken string) ([]*Email, error) {
	var emails []*Email
	err := a.Get("/user/emails", nil, accessToken, &emails)
	return emails, err
}

func (a *App) Repositories(username string, accessToken string) ([]*Repository, error) {
	var repos []*Repository
	var p string
	if username == "" {
		p = "/user/repos"
	} else {
		p = fmt.Sprintf("/users/%s/repos", username)
	}
	err := a.Get(p, nil, accessToken, &repos)
	return repos, err
}

func (a *App) Repository(username string, name string, accessToken string) (*Repository, error) {
	var repo *Repository
	p := fmt.Sprintf("/repos/%s/%s", username, name)
	err := a.Get(p, nil, accessToken, &repo)
	return repo, err
}

func (a *App) Branches(repo *Repository, accessToken string) ([]*Branch, error) {
	var branches []*Branch
	p := repo.URL + "/branches"
	err := a.Get(p, nil, accessToken, &branches)
	return branches, err
}

func (a *App) Tags(repo *Repository, accessToken string) ([]*Tag, error) {
	var tags []*Tag
	p := repo.URL + "/tags"
	err := a.Get(p, nil, accessToken, &tags)
	return tags, err
}

func decodeError(r *httpclient.Response) error {
	var m map[string]interface{}
	r.UnmarshalJSON(&m)
	message, _ := m["message"].(string)
	return errors.New(message)
}
