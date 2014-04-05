package github

import (
	"fmt"
)

type User struct {
	Login      string `json:"login"`
	Type       string `json:"type"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Company    string `json:"company"`
	Location   string `json:"location"`
	Blog       string `json:"blog"`
	Avatar     string `json:"avatar_url"`
	GravatarId string `json:"gravatar_id"`
	Url        string `json:"html_url"`
}

type UserResource struct {
	client *Client
}

func (r *UserResource) Current() (*User, error) {
	// we're gonna parse this data as a hash map because
	// github returns nil for many of these values, and
	// as of go1, the parser chokes on a value of null.
	m := map[string]interface{}{}
	if err := r.client.do("GET", "/user", nil, &m); err != nil {
		return nil, err
	}

	user := mapToUser(m)
	if len(user.Login) == 0 {
		return nil, ErrNotFound
	}

	return user, nil
}

func (r *UserResource) Find(username string) (*User, error) {
	m := map[string]interface{}{}
	path := fmt.Sprintf("/users/%s", username)
	if err := r.client.do("GET", path, nil, &m); err != nil {
		return nil, err
	}

	user := mapToUser(m)
	if len(user.Login) == 0 {
		return nil, ErrNotFound
	}

	return user, nil
}

// -----------------------------------------------------------------------------
// Helper functions converting map[string]interface{} to a User

func mapToUser(m map[string]interface{}) *User {
	user := User { }
	for k, v := range m {
		// ignore nil values
		if v == nil {
			continue
		}

		// ignore non-string values
		var str string
		var ok bool
		if str, ok = v.(string); !ok {
			continue
		} 

		switch k {
		case "login"       : user.Login = str
		case "type"        : user.Type = str
		case "name"        : user.Name = str
		case "email"       : user.Email = str
		case "company"     : user.Company = str
		case "location"    : user.Location = str
		case "avatar_url"  : user.Avatar = str
		case "gravatar_id" : user.GravatarId = str
		case "html_url"    : user.Url = str
		}
	}

	return &user
}
