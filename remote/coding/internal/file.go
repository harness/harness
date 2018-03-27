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
	"fmt"
)

type Commit struct {
	File *File `json:"file"`
}

type File struct {
	Data string `json:"data"`
}

func (c *Client) GetFile(globalKey, projectName, ref, path string) ([]byte, error) {
	u := fmt.Sprintf("/user/%s/project/%s/git/blob/%s/%s", globalKey, projectName, ref, path)
	resp, err := c.Get(u, nil)
	if err != nil {
		return nil, err
	}
	commit := &Commit{}
	err = json.Unmarshal(resp, commit)
	if err != nil {
		return nil, APIClientErr{"fail to parse file data", u, err}
	}
	if commit == nil || commit.File == nil {
		return nil, nil
	}
	return []byte(commit.File.Data), nil
}
