package user

import (
	"testing"
)

func TestSetEmail(t *testing.T) {
	user := User{}
	user.SetEmail("winkle@caltech.edu")

	// make sure the email was correctly set
	var got, want = user.Email, "winkle@caltech.edu"
	if got != want {
		t.Errorf("Want Email %s, got %s", want, got)
	}

	// make sure the gravatar hash was correctly calculated
	got, want = user.Gravatar, "ab23a88a3ed77ecdfeb894c0eaf2817a"
	if got != want {
		t.Errorf("Want Gravatar %s, got %s", want, got)
	}
}
