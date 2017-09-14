package client

import (
	"encoding/json"
)

const (
	currentUserUrl = "/user"
)

func (c *Client) CurrentUser() (User, error) {
	url, opaque := c.ResourceUrl(currentUserUrl, nil, nil)
	var user User

	contents, err := c.Do("GET", url, opaque, nil)
	if err == nil {
		err = json.Unmarshal(contents, &user)
	}

	return user, err
}
