package client

import (
	"encoding/json"
)

const (
	currentUserUrl = "/api/user.whoami"
)

func (c *Client) CurrentUser() (User, error) {
	var user User

	url, opaque := c.ResourceUrl(currentUserUrl, nil, nil)
	contents, err := c.Do("GET", url, opaque, nil)

	if err == nil {
		err = json.Unmarshal(contents, &user)
	}

	return user, err
}
