package bitbucket

import (
	"fmt"
	"net/url"
)

// An account can have one or more email addresses associated with it.
// Use this end point to list, change, or create an email address.
//
// https://confluence.atlassian.com/display/BITBUCKET/emails+Resource
type EmailResource struct {
	client *Client
}

type Email struct {
	// Indicates the user confirmed the email address (true).
	Active bool `json:"active"`

	// The email address.
	Email string `json:"email"`

	// Indicates the email is the main contact email address for the account.
	Primary bool `json:"primary"`
}

// Gets the email addresses associated with the account. This call requires
// authentication.
func (r *EmailResource) List(account string) ([]*Email, error) {
	emails := []*Email{}
	path := fmt.Sprintf("/users/%s/emails", account)

	if err := r.client.do("GET", path, nil, nil, &emails); err != nil {
		return nil, err
	}

	return emails, nil
}

// Gets an individual email address associated with an account.
// This call requires authentication.
func (r *EmailResource) Find(account, address string) (*Email, error) {
	email := Email{}
	path := fmt.Sprintf("/users/%s/emails/%s", account, address)

	if err := r.client.do("GET", path, nil, nil, &email); err != nil {
		return nil, err
	}

	return &email, nil
}

// Gets an individual's primary email address.
func (r *EmailResource) FindPrimary(account string) (*Email, error) {
	emails, err := r.List(account)
	if err != nil {
		return nil, err
	}

	for _, email := range emails {
		if email.Primary {
			return email, nil
		}
	}

	return nil, ErrNotFound
}

// Adds additional email addresses to an account. This call requires
// authentication.
func (r *EmailResource) Create(account, address string) (*Email, error) {

	values := url.Values{}
	values.Add("email", address)

	e := Email{}
	path := fmt.Sprintf("/users/%s/emails/%s", account, address)
	if err := r.client.do("POST", path, nil, values, &e); err != nil {
		return nil, err
	}

	return &e, nil
}
