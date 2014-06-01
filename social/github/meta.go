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
