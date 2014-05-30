package facebook

import (
	"time"
)

type Error struct {
	Type    string
	Message string
}

func (e Error) Error() string {
	return e.Message
}

type ErrorContainer struct {
	Error Error
}

type Comment struct {
	Id        string
	From      string
	FromId    string
	Message   string
	Created   time.Time
	LikeCount int
	Comments  []*Comment /* replies */
}
