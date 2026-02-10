//  Copyright 2023 Harness, Inc.
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

package auth

import (
	"net/http"
	"strings"
)

// AccessSet maps a typed, named resource to
// a set of actions requested or authorized.
type AccessSet map[Resource]ActionSet

// NewAccessSet constructs an accessSet from
// a variable number of auth.Access items.
func NewAccessSet(accessItems ...Access) AccessSet {
	accessSet := make(AccessSet, len(accessItems))

	for _, access := range accessItems {
		resource := Resource{
			Type: access.Type,
			Name: access.Name,
		}

		set, exists := accessSet[resource]
		if !exists {
			set = NewActionSet()
			accessSet[resource] = set
		}

		set.Add(access.Action)
	}

	return accessSet
}

// Contains returns whether or not the given access is in this accessSet.
func (s AccessSet) Contains(access Access) bool {
	actionSet, ok := s[access.Resource]
	if ok {
		return actionSet.contains(access.Action)
	}

	return false
}

// ScopeParam returns a collection of scopes which can
// be used for a WWW-Authenticate challenge parameter.
func (s AccessSet) ScopeParam() string {
	scopes := make([]string, 0, len(s))

	for resource, actionSet := range s {
		actions := strings.Join(actionSet.keys(), ",")
		resourceName := resource.Name
		scopes = append(scopes, strings.Join([]string{resource.Type, resourceName, actions}, ":"))
	}

	return strings.Join(scopes, " ")
}

// Resource describes a resource by type and name.
type Resource struct {
	Type string
	Name string
}

// Access describes a specific action that is
// requested or allowed for a given resource.
type Access struct {
	Resource
	Action string
}

func AppendAccess(records []Access, method string, name string) []Access {
	resource := Resource{
		Type: "repository",
		Name: name,
	}

	switch method {
	case http.MethodGet, http.MethodHead:
		records = append(records,
			Access{
				Resource: resource,
				Action:   "pull",
			})
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		records = append(records,
			Access{
				Resource: resource,
				Action:   "pull",
			},
			Access{
				Resource: resource,
				Action:   "push",
			})
	case http.MethodDelete:
		records = append(records,
			Access{
				Resource: resource,
				Action:   "delete",
			})
	}
	return records
}

// ActionSet is a special type of stringSet.
type ActionSet struct {
	stringSet
}

func NewActionSet(actions ...string) ActionSet {
	return ActionSet{newStringSet(actions...)}
}

// Contains calls StringSet.Contains() for
// either "*" or the given action string.
func (s ActionSet) Contains(action string) bool {
	return s.stringSet.contains("*") || s.stringSet.contains(action)
}

// StringSet is a useful type for looking up strings.
type stringSet map[string]struct{}

// NewStringSet creates a new StringSet with the given strings.
func newStringSet(keys ...string) stringSet {
	ss := make(stringSet, len(keys))
	ss.Add(keys...)
	return ss
}

// Add inserts the given keys into this StringSet.
func (ss stringSet) Add(keys ...string) {
	for _, key := range keys {
		ss[key] = struct{}{}
	}
}

// Contains returns whether the given key is in this StringSet.
func (ss stringSet) contains(key string) bool {
	_, ok := ss[key]
	return ok
}

// Keys returns a slice of all keys in this StringSet.
func (ss stringSet) keys() []string {
	keys := make([]string, 0, len(ss))

	for key := range ss {
		keys = append(keys, key)
	}

	return keys
}

func HasWriteOrDeleteScope(scope string) bool {
	scopes := strings.Fields(scope)
	for _, s := range scopes {
		parts := strings.Split(s, ":")
		if len(parts) >= 3 {
			actions := strings.Split(parts[2], ",")
			for _, action := range actions {
				if action == "push" || action == "delete" || action == "*" {
					return true
				}
			}
		}
	}
	return false
}
