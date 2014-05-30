package facebook

import (
	"errors"
	"fmt"
)

var (
	errInvalidDataFormat = errors.New("invalid data format")
)

func parseComments(root map[string]interface{}) (comments []*Comment, err error) {
	defer func() {
		if r := recover(); r != nil {
			comments = nil
			err = errInvalidDataFormat
		}
	}()
	comm, ok := root["comments"].(map[string]interface{})
	if !ok {
		return nil, errInvalidDataFormat
	}
	data, ok := comm["data"].([]interface{})
	if !ok {
		return nil, errInvalidDataFormat
	}
	for _, v := range data {
		c, err := parseComment(v)
		if err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, nil
}

func parseComment(data interface{}) (*Comment, error) {
	c := data.(map[string]interface{})
	from := c["from"].(map[string]interface{})
	author := from["name"].(string)
	authorId := from["id"].(string)
	message := c["message"].(string)
	/* replies lack like_count */
	likeCount, _ := c["like_count"].(float64)
	cr := c["created_time"].(string)
	created, err := parseFacebookTime(cr)
	if err != nil {
		return nil, err
	}
	comment := &Comment{
		From:      author,
		FromId:    authorId,
		Message:   message,
		Created:   created,
		LikeCount: int(likeCount),
	}
	comments, _ := parseComments(c)
	if comments != nil {
		comment.Comments = comments
	}
	return comment, nil
}

func (a *App) Comments(url string) ([]*Comment, error) {
	commentsUrl := fmt.Sprintf("http://graph.facebook.com/comments/?ids=%s", url)
	resp, err := a.client().HTTPClient.Get(commentsUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	var value interface{}
	if err := resp.DecodeJSON(&value); err != nil {
		return nil, err
	}
	dataMap, ok := value.(map[string]interface{})
	if !ok {
		return nil, errInvalidDataFormat
	}
	root, _ := dataMap[url].(map[string]interface{})
	if root == nil {
		return nil, errInvalidDataFormat
	}
	return parseComments(root)
}
