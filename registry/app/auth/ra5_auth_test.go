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

package auth

import (
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	ra5Repo = "library/app"
	ra5Pull = "pull"
	ra5Push = "push"
)

func TestRa5AppendAccess(t *testing.T) {
	tests := []struct {
		method  string
		actions []string
	}{
		{http.MethodGet, []string{ra5Pull}},
		{http.MethodHead, []string{ra5Pull}},
		{http.MethodPost, []string{ra5Pull, ra5Push}},
		{http.MethodPut, []string{ra5Pull, ra5Push}},
		{http.MethodPatch, []string{ra5Pull, ra5Push}},
		{http.MethodDelete, []string{"delete"}},
		{http.MethodOptions, nil},
	}
	for _, tc := range tests {
		t.Run(tc.method, func(t *testing.T) {
			got := AppendAccess(nil, tc.method, ra5Repo)
			actions := make([]string, 0, len(got))
			for _, a := range got {
				require.Equal(t, "repository", a.Type)
				require.Equal(t, ra5Repo, a.Name)
				actions = append(actions, a.Action)
			}
			if tc.actions == nil {
				require.Empty(t, actions)
				return
			}
			require.Equal(t, tc.actions, actions)
		})
	}
}

func TestRa5AccessSet(t *testing.T) {
	pull := Access{Resource: Resource{Type: "repository", Name: ra5Repo}, Action: ra5Pull}
	push := Access{Resource: pull.Resource, Action: ra5Push}
	other := Access{Resource: Resource{Type: "repository", Name: "other"}, Action: ra5Pull}

	set := NewAccessSet(pull, push, other)
	require.Len(t, set, 2)
	require.True(t, set.Contains(pull))
	require.True(t, set.Contains(push))
	require.False(t, set.Contains(Access{Resource: pull.Resource, Action: "delete"}))
	require.False(t, set.Contains(Access{Resource: Resource{Type: "repository", Name: "none"}, Action: ra5Pull}))

	scopes := strings.Fields(set.ScopeParam())
	sort.Strings(scopes)
	require.Len(t, scopes, 2)
	require.Equal(t, "repository:other:pull", scopes[1])
	require.True(t, strings.HasPrefix(scopes[0], "repository:library/app:"))
	require.Contains(t, scopes[0], ra5Pull)
	require.Contains(t, scopes[0], ra5Push)

	require.Empty(t, NewAccessSet().ScopeParam())
}

func TestRa5ActionSet(t *testing.T) {
	require.True(t, NewActionSet(ra5Pull).Contains(ra5Pull))
	require.False(t, NewActionSet(ra5Pull).Contains(ra5Push))
	require.True(t, NewActionSet("*").Contains("anything"))
	require.False(t, NewActionSet().Contains(ra5Pull))
}

func TestRa5HasWriteOrDeleteScope(t *testing.T) {
	tests := []struct {
		scope string
		want  bool
	}{
		{"", false},
		{"repository:a:pull", false},
		{"repository:a:pull,push", true},
		{"repository:a:delete", true},
		{"repository:a:*", true},
		{"repository:a:pull repository:b:push", true},
		{"repository:a", false},
		{"repository:a:pushing", false},
	}
	for _, tc := range tests {
		t.Run(tc.scope, func(t *testing.T) {
			require.Equal(t, tc.want, HasWriteOrDeleteScope(tc.scope))
		})
	}
}
