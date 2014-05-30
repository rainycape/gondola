package google

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type Value struct {
	Value string
	Type  string
	Label string
}

type AgeRange struct {
	Min int
	Max int
}

type Name struct {
	Display string
	Given   string
	Family  string
}

type Person struct {
	Id         string
	Name       Name
	Image      string
	URL        string
	Emails     []*Value
	Links      []*Value
	Gender     string
	Occupation string
	Age        AgeRange
	Lang       string
}

func (p *Person) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	var m map[string]interface{}
	if err := dec.Decode(&m); err != nil {
		return err
	}
	return p.decodeMap(m)
}

func (p *Person) decodeMap(m map[string]interface{}) error {
	p.Id = m["id"].(string)
	name := m["name"].(map[string]interface{})
	p.Name = Name{
		Display: m["displayName"].(string),
		Given:   name["givenName"].(string),
		Family:  name["familyName"].(string),
	}
	image := m["image"].(map[string]interface{})
	p.Image = image["url"].(string)
	p.URL = m["url"].(string)
	p.Emails = decodeValues(m["emails"])
	p.Links = decodeValues(m["urls"])
	p.Gender, _ = m["gender"].(string)
	p.Occupation, _ = m["occupation"].(string)
	if age, ok := m["ageRange"].(map[string]interface{}); ok {
		min, _ := age["min"].(float64)
		max, _ := age["max"].(float64)
		p.Age = AgeRange{
			Min: int(min),
			Max: int(max),
		}
	}
	p.Lang = m["language"].(string)
	return nil
}

func decodeValues(v interface{}) []*Value {
	vv, ok := v.([]interface{})
	if ok {
		values := make([]*Value, len(vv))
		for ii, v := range vv {
			m := v.(map[string]interface{})
			value, _ := m["value"].(string)
			typ, _ := m["type"].(string)
			label, _ := m["label"].(string)
			values[ii] = &Value{
				Value: value,
				Type:  typ,
				Label: label,
			}
		}
		return values
	}
	return nil
}

func (a *App) Person(id string, accessToken string) (*Person, error) {
	p := fmt.Sprintf("https://www.googleapis.com/plus/v1/people/%s", id)
	resp, err := a.client().Get(p, nil, accessToken)
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	var person *Person
	if err := resp.UnmarshalJSON(&person); err != nil {
		return nil, err
	}
	return person, nil

}
