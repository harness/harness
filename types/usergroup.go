// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package types defines common data structures.
package types

type UserGroup struct {
	ID          int64    `json:"-"`
	Identifier  string   `json:"identifier"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	SpaceID     int64    `json:"-"`
	Created     int64    `json:"created"`
	Updated     int64    `json:"updated"`
	Users       []string // Users are used by the code owners code
}

type UserGroupInfo struct {
	ID          int64  `json:"id"`
	Identifier  string `json:"identifier"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (u *UserGroup) ToUserGroupInfo() *UserGroupInfo {
	return &UserGroupInfo{
		ID:          u.ID,
		Identifier:  u.Identifier,
		Name:        u.Name,
		Description: u.Description,
	}
}
