// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
