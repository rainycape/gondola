package github

type Commit struct {
	Sha1 string `json:"sha1"`
	URL  string `json:"url"`
}

type Branch struct {
	Name   string `json:"name"`
	Commit Commit `json:"commit"`
}

type Tag struct {
	Name       string `json:"name"`
	Commit     Commit `json:"commit"`
	ZipballURL string `json:"zipball_url"`
	TarballURL string `json:"tarball_url"`
}

type Email struct {
	Address  string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

type Organization struct {
	Id        int64  `json:"id"`
	Login     string `json:"login"`
	URL       string `json:"url"`
	AvatarURL string `json:"avatar_url"`
}
