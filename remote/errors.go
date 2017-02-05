package remote

// AuthError represents remote authentication error.
type AuthError struct {
	Err         string
	Description string
	URI         string
}

// Error implements error interface.
func (ae *AuthError) Error() string {
	err := ae.Err
	if ae.Description != "" {
		err += " " + ae.Description
	}
	if ae.URI != "" {
		err += " " + ae.URI
	}
	return err
}

// check interface
var _ error = new(AuthError)
