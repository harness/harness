package internal

import (
	"encoding/json"
)

type User struct {
	GlobalKey string `json:"global_key"`
	Email     string `json:"email"`
	Avatar    string `json:"avatar"`
}

func (c *Client) GetCurrentUser() (*User, error) {
	u := "/account/current_user"
	resp, err := c.Get(u, nil)
	if err != nil {
		return nil, err
	}
	user := &User{}
	err = json.Unmarshal(resp, user)
	if err != nil {
		return nil, APIClientErr{"fail to parse current user data", u, err}
	}
	return user, nil
}
