package social

type Service int

const (
	Facebook Service = iota + 1
	Twitter
)

func (s Service) String() string {
	switch s {
	case Facebook:
		return "Facebook"
	case Twitter:
		return "Twitter"
	}
	return ""
}
