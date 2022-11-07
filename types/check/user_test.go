// Copyright 2022 Harness Inc. All rights reserved.
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
		uid         string
		email       string
		displayName string
		error       error
	}{
		{
			uid:         "jane",
			email:       "jane@gmail.com",
			displayName: "Jane Smith",
		},
	}
	for _, test := range tests {
		user := &types.User{UID: test.uid, Email: test.email, DisplayName: test.displayName}
		err := UserDefault(user)
		if got, want := err, test.error; !errors.Is(got, want) {
			t.Errorf("Want user %s error %v, got %v", test.email, want, got)
		}
	}
}
