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

package request

import (
	"net/http"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	PathParamGitspaceIdentifier = "gitspace_identifier"
	QueryParamGitspaceOwner     = "gitspace_owner"
	QueryParamGitspaceStates    = "gitspace_states"
	QueryParamOrgs              = "org_identifiers"
	QueryParamProjects          = "project_identifiers"
)

func GetGitspaceRefFromPath(r *http.Request) (string, error) {
	return PathParamOrError(r, PathParamGitspaceIdentifier)
}

// ParseGitspaceSort extracts the gitspace sort parameter from the url.
func ParseGitspaceSort(r *http.Request) enum.GitspaceSort {
	return enum.ParseGitspaceSort(
		r.URL.Query().Get(QueryParamSort),
	)
}

// ParseGitspaceOwner extracts the gitspace owner type from the url.
func ParseGitspaceOwner(r *http.Request) enum.GitspaceOwner {
	return enum.ParseGitspaceOwner(
		r.URL.Query().Get(QueryParamGitspaceOwner),
	)
}

// ParseGitspaceStates extracts the gitspace owner type from the url.
func ParseGitspaceStates(r *http.Request) []enum.GitspaceFilterState {
	pTypesRaw := r.URL.Query()[QueryParamGitspaceStates]
	m := make(map[enum.GitspaceFilterState]struct{}) // use map to eliminate duplicates
	for _, pTypeRaw := range pTypesRaw {
		if pType, ok := enum.GitspaceFilterState(pTypeRaw).Sanitize(); ok {
			m[pType] = struct{}{}
		}
	}

	res := make([]enum.GitspaceFilterState, 0, len(m))
	for t := range m {
		res = append(res, t)
	}

	return res
}

// ParseScopeFilter extracts scope filter from the url.
func ParseScopeFilter(r *http.Request) types.ScopeFilter {
	orgsTypesRaw := r.URL.Query()[QueryParamOrgs]
	orgs := make([]string, 0, len(orgsTypesRaw))
	orgsSet := make(map[string]struct{})
	for _, pTypeRaw := range orgsTypesRaw {
		if _, ok := orgsSet[pTypeRaw]; ok {
			// already added
			continue
		}
		orgs = append(orgs, pTypeRaw)
		orgsSet[pTypeRaw] = struct{}{}
	}

	projectsTypesRaw := r.URL.Query()[QueryParamProjects]
	projects := make([]string, 0, len(projectsTypesRaw))
	projectsSet := make(map[string]struct{})
	for _, pTypeRaw := range projectsTypesRaw {
		if _, ok := projectsSet[pTypeRaw]; ok {
			// already added
			continue
		}
		projects = append(projects, pTypeRaw)
		projectsSet[pTypeRaw] = struct{}{}
	}

	return types.ScopeFilter{
		Orgs:     orgs,
		Projects: projects,
	}
}

// ParseGitspaceFilter extracts the gitspace filter from the url.
func ParseGitspaceFilter(r *http.Request) types.GitspaceFilter {
	return types.GitspaceFilter{
		QueryFilter:          ParseListQueryFilterFromRequest(r),
		GitspaceFilterStates: ParseGitspaceStates(r),
		Sort:                 ParseGitspaceSort(r),
		Owner:                ParseGitspaceOwner(r),
		Order:                ParseOrder(r),
		ScopeFilter:          ParseScopeFilter(r),
	}
}
