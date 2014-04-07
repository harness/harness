package bitbucket

import (
	"fmt"
)

type Account struct {
	User  *User   `json:"user"`
	Repos []*Repo `json:"repositories"`
}

type User struct {
	Username    string `json:"username"`   // The name associated with the account.
	FirstName   string `json:"first_name"` //	The first name associated with account.
	LastName    string `json:"last_name"`  // The last name associated with the account. For a team account, this value is always empty.
	DisplayName string `json:"display_name"`
	Avatar      string `json:"avatar"`  // An avatar associated with the account.
	IsTeam      bool   `json:"is_team"` // Indicates if this is a Team account.

}

// Use the /user endpoints to gets information related to a user
// or team account
//
// https://confluence.atlassian.com/display/BITBUCKET/user+Endpoint
type UserResource struct {
	client *Client
}

// Gets the basic information associated with an account and a list
// of all its repositories both public and private.
func (r *UserResource) Current() (*Account, error) {
	user := Account{}
	if err := r.client.do("GET", "/user", nil, nil, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// Gets the basic information associated with the specified user
// account.
func (r *UserResource) Find(username string) (*Account, error) {
	user := Account{}
	path := fmt.Sprintf("/users/%s", username)

	if err := r.client.do("GET", path, nil, nil, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

/* TODO 
// Updates the basic information associated with an account.
// It operates on the currently authenticated user.
func (r *UserResource) Update(user *User) (*User, error) {
	return nil, nil
}
*/
