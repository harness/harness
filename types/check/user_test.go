// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"errors"
	"testing"

	"github.com/harness/gitness/types"
)

func TestUser(t *testing.T) {
	tests := []struct {
		email string
		error error
	}{
		{
			email: "jane@gmail.com",
		},
	}
	for _, test := range tests {
		user := &types.User{Email: test.email}
		err := User(user)
		if got, want := err, test.error; !errors.Is(got, want) {
			t.Errorf("Want user %s error %v, got %v", test.email, want, got)
		}
	}
}
