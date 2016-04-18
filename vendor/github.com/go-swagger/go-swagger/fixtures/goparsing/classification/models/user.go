// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import "github.com/go-swagger/go-swagger/strfmt"

// User represents the user for this application
//
// A user is the security principal for this aplication.
// It's also used as one of main axis for reporting.
//
// A user can have friends with whom they can share what they like.
//
// swagger:model
type User struct {
	// the id for this user
	//
	// required: true
	// min: 1
	ID int64 `json:"id"`

	// the name for this user
	// required: true
	// min length: 3
	Name string `json:"name"`

	// the email address for this user
	//
	// required: true
	// unique: true
	Email strfmt.Email `json:"login"`

	// the friends for this user
	Friends []User `json:"friends"`
}
