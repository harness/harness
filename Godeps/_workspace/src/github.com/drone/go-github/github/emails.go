package github

// An account can have one or more email addresses associated with it.
// Use this end point to list, change, or create an email address.
type EmailResource struct {
	client *Client
}

type Email struct {
	// Indicates the user confirmed the email address (true).
	Verified bool `json:"verified"`

	// The email address.
	Email string `json:"email"`

	// Indicates the email is the main contact email address for the account.
	Primary bool `json:"primary"`
}

// Gets the email addresses associated with the account.
func (r *EmailResource) List() ([]*Email, error) {
	emails := []*Email{}
	if err := r.client.do("GET", "/user/emails", nil, &emails); err != nil {
		return nil, err
	}

	return emails, nil
}

// Gets an individual email address associated with an account.
func (r *EmailResource) Find(address string) (*Email, error) {
	emails, err := r.List()
	if err != nil {
		return nil, err
	}

	for _, email := range emails {
		if email.Email == address {
			return email, nil
		}
	}

	return nil, ErrNotFound
}

// Gets an individual's primary email address.
func (r *EmailResource) FindPrimary() (*Email, error) {
	emails, err := r.List()
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

// Adds additional email addresses to an account.
func (r *EmailResource) Create(address string) (error) {
	emails := []string{ address }
	if err := r.client.do("POST", "/user/emails", &emails, nil); err != nil {
		return err
	}

	return nil
}

// Deletes an email addresses from an account.
func (r *EmailResource) Delete(address string) (error) {
	emails := []string{ address }
	if err := r.client.do("DELETE", "/user/emails", &emails, nil); err != nil {
		return err
	}

	return nil
}
