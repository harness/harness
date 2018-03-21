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
	"net/url"
	"strings"
)

var encodeMap = map[string]string{
	".": "%252E",
}

func encodeParameter(value string) string {
	value = url.QueryEscape(value)

	for before, after := range encodeMap {
		value = strings.Replace(value, before, after, -1)
	}

	return value
}

// Tag returns current tag for push event hook payload
// This function returns empty string for any other events
func (h *HookPayload) Tag() string {
	return strings.TrimPrefix(h.Ref, "refs/tags/")
}

// Branch returns current branch for push event hook payload
// This function returns empty string for any other events
func (h *HookPayload) Branch() string {
	return strings.TrimPrefix(h.Ref, "refs/heads/")
}

// Head returns the latest changeset for push event hook payload
func (h *HookPayload) Head() hCommit {
	c := hCommit{}
	for _, cm := range h.Commits {
		if h.After == cm.Id {
			return cm
		}
	}
	return c
}
