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

package client

import (
	"encoding/json"
	"strconv"
)

const (
	groupsUrl = "/groups"
)

// Get a list of all projects owned by the authenticated user.
func (g *Client) AllGroups() ([]*Namespace, error) {
	var perPage = 100
	var groups []*Namespace

	for i := 1; true; i++ {
		contents, err := g.Groups(i, perPage)
		if err != nil {
			return groups, err
		}

		for _, value := range contents {
			groups = append(groups, value)
		}

		if len(groups) == 0 {
			break
		}

		if len(groups)/i < perPage {
			break
		}
	}

	return groups, nil
}

func (g *Client) Groups(page, perPage int) ([]*Namespace, error) {
	url, opaque := g.ResourceUrl(groupsUrl, nil, QMap{
		"page":     strconv.Itoa(page),
		"per_page": strconv.Itoa(perPage),
	})

	var groups []*Namespace

	contents, err := g.Do("GET", url, opaque, nil)
	if err == nil {
		err = json.Unmarshal(contents, &groups)
	}

	return groups, err
}
