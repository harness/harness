package model

import "testing"

func TestUserValidate(t *testing.T) {
	var tests = []struct {
		user User
		err  error
	}{
		{
			user: User{},
			err:  errUserLoginInvalid,
		},
		{
			user: User{Login: "octocat!"},
			err:  errUserLoginInvalid,
		},
		{
			user: User{Login: "!octocat"},
			err:  errUserLoginInvalid,
		},
		{
			user: User{Login: "john$smith"},
			err:  errUserLoginInvalid,
		},
		{
			user: User{Login: "octocat"},
			err:  nil,
		},
		{
			user: User{Login: "john-smith"},
			err:  nil,
		},
		{
			user: User{Login: "john_smith"},
			err:  nil,
		},
		{
			user: User{Login: "john.smith"},
			err:  nil,
		},
	}

	for _, test := range tests {
		err := test.user.Validate()
		if want, got := test.err, err; want != got {
			t.Errorf("Want user validation error %s, got %s", want, got)
		}
	}
}
