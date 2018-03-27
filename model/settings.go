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

// Settings defines system configuration parameters.
type Settings struct {
	Open   bool            // Enables open registration
	Secret string          // Secret token used to authenticate agents
	Admins map[string]bool // Administrative users
	Orgs   map[string]bool // Organization whitelist
}

// IsAdmin returns true if the user is a member of the administrator list.
func (c *Settings) IsAdmin(user *User) bool {
	return c.Admins[user.Login]
}

// IsMember returns true if the user is a member of the whitelisted teams.
func (c *Settings) IsMember(teams []*Team) bool {
	for _, team := range teams {
		if c.Orgs[team.Login] {
			return true
		}
	}
	return false
}
