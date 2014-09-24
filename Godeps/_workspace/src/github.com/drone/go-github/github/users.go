package github

import (
	"fmt"
)

type User struct {
	ID         int64  `json:"id"`
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

		switch k {
		case "login"       : user.Login, _ = v.(string)
		case "type"        : user.Type, _ = v.(string)
		case "name"        : user.Name, _ = v.(string)
		case "email"       : user.Email, _ = v.(string)
		case "company"     : user.Company, _ = v.(string)
		case "location"    : user.Location, _ = v.(string)
		case "avatar_url"  : user.Avatar, _ = v.(string)
		case "gravatar_id" : user.GravatarId, _ = v.(string)
		case "html_url"    : user.Url, _ = v.(string)
		case "id"          :
			if id, ok := v.(float64); ok {
				user.ID = int64(id)
			}
		}
	}

	return &user
}

