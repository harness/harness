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

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	PathParamPublicKeyIdentifier = "public_key_identifier"

	QueryParamPublicKeyScheme = "public_key_scheme"
	QueryParamPublicKeyUsage  = "public_key_usage"
)

func GetPublicKeyIdentifierFromPath(r *http.Request) (string, error) {
	return PathParamOrError(r, PathParamPublicKeyIdentifier)
}

// ParsePublicKeyScheme extracts the public key scheme from the url.
func ParsePublicKeyScheme(r *http.Request) []enum.PublicKeyScheme {
	strSchemeList, _ := QueryParamList(r, QueryParamPublicKeyScheme)
	m := make(map[enum.PublicKeyScheme]struct{}) // use map to eliminate duplicates
	for _, s := range strSchemeList {
		if state, ok := enum.PublicKeyScheme(s).Sanitize(); ok {
			m[state] = struct{}{}
		}
	}

	schemeList := make([]enum.PublicKeyScheme, 0, len(m))
	for s := range m {
		schemeList = append(schemeList, s)
	}

	return schemeList
}

// ParsePublicKeyUsage extracts the public key usage from the url.
func ParsePublicKeyUsage(r *http.Request) []enum.PublicKeyUsage {
	strUsageList, _ := QueryParamList(r, QueryParamPublicKeyUsage)
	m := make(map[enum.PublicKeyUsage]struct{}) // use map to eliminate duplicates
	for _, s := range strUsageList {
		if state, ok := enum.PublicKeyUsage(s).Sanitize(); ok {
			m[state] = struct{}{}
		}
	}

	usageList := make([]enum.PublicKeyUsage, 0, len(m))
	for s := range m {
		usageList = append(usageList, s)
	}

	return usageList
}

// ParseListPublicKeyQueryFilterFromRequest parses query filter for public keys from the url.
func ParseListPublicKeyQueryFilterFromRequest(r *http.Request) (types.PublicKeyFilter, error) {
	sort := enum.PublicKeySort(ParseSort(r))
	sort, ok := sort.Sanitize()
	if !ok {
		return types.PublicKeyFilter{}, usererror.BadRequest("Invalid value for the sort query parameter.")
	}

	schemes := ParsePublicKeyScheme(r)
	usages := ParsePublicKeyUsage(r)

	return types.PublicKeyFilter{
		ListQueryFilter: ParseListQueryFilterFromRequest(r),
		Sort:            sort,
		Order:           ParseOrder(r),
		Usages:          usages,
		Schemes:         schemes,
	}, nil
}
