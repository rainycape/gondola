package facebook

type PictureData struct {
	IsSilhouette bool   `json:"is_silhouette"`
	URL          string `json:"url"`
}

type Picture struct {
	Data *PictureData `json:"data"`
}

type Person struct {
	Id          string   `json:"id"`
	FirstName   string   `json:"first_name"`
	Gender      string   `json:"gender"`
	LastName    string   `json:"last_name"`
	Link        string   `json:"link"`
	Locale      string   `json:"locale"`
	MiddleName  string   `json:"middle_name"`
	Name        string   `json:"name"`
	Timezone    float64  `json:"timezone"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	UpdatedTime string   `json:"updated_time"`
	Verified    bool     `json:"verified"`
	Picture     *Picture `json:"picture"`
}
