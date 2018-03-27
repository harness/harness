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

package internal

import (
	"encoding/json"
)

type User struct {
	GlobalKey string `json:"global_key"`
	Email     string `json:"email"`
	Avatar    string `json:"avatar"`
}

func (c *Client) GetCurrentUser() (*User, error) {
	u := "/account/current_user"
	resp, err := c.Get(u, nil)
	if err != nil {
		return nil, err
	}
	user := &User{}
	err = json.Unmarshal(resp, user)
	if err != nil {
		return nil, APIClientErr{"fail to parse current user data", u, err}
	}
	return user, nil
}
