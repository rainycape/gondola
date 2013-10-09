package facebook

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
)

func parseComments(root map[string]interface{}) []*Comment {
	comm, ok := root["comments"].(map[string]interface{})
	if !ok {
		return nil
	}
	data, ok := comm["data"].([]interface{})
	if !ok {
		return nil
	}
	var comments []*Comment
	for _, v := range data {
		c := parseComment(v)
		if c != nil {
			comments = append(comments, c)
		}
	}
	return comments
}

func parseComment(data interface{}) *Comment {
	c := data.(map[string]interface{})
	from := c["from"].(map[string]interface{})
	author := from["name"].(string)
	authorId := from["id"].(string)
	message := c["message"].(string)
	/* replies lack like_count */
	likeCount, _ := c["like_count"].(float64)
	cr := c["created_time"].(string)
	created, err := ParseFacebookTime(cr)
	if err != nil {
		return nil
	}
	comment := &Comment{
		From:      author,
		FromId:    authorId,
		Message:   message,
		Created:   created,
		LikeCount: int(likeCount),
	}
	comments := parseComments(c)
	if comments != nil {
		comment.Comments = comments
	}
	return comment
}

func FetchComments(url string) (comments []*Comment, err error) {
	defer func() {
		if r := recover(); r != nil && comments == nil && err == nil {
			_, file, line, ok := runtime.Caller(4)
			if !ok {
				file = "???"
				line = 0
			}
			err = errors.New(fmt.Sprintf("Invalid data format: %s at %s:%d", r, file, line))
		}
	}()
	commentsUrl := fmt.Sprintf("http://graph.facebook.com/comments/?ids=%s", url)
	resp, err := http.Get(commentsUrl)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	var value interface{}
	err = decoder.Decode(&value)
	if err != nil {
		return
	}
	dataMap, ok := value.(map[string]interface{})
	if !ok {
		return
	}
	root, _ := dataMap[url].(map[string]interface{})
	if root == nil {
		return
	}
	comments = parseComments(root)
	return
}
