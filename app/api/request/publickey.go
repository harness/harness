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
	"net/url"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	PathParamPublicKeyIdentifier = "public_key_identifier"
)

func GetPublicKeyIdentifierFromPath(r *http.Request) (string, error) {
	identifier, err := PathParamOrError(r, PathParamPublicKeyIdentifier)
	if err != nil {
		return "", err
	}

	// paths are unescaped
	return url.PathUnescape(identifier)
}

// ParseListPublicKeyQueryFilterFromRequest parses query filter for public keys from the url.
func ParseListPublicKeyQueryFilterFromRequest(r *http.Request) (types.PublicKeyFilter, error) {
	sort := enum.PublicKeySort(ParseSort(r))
	sort, ok := sort.Sanitize()
	if !ok {
		return types.PublicKeyFilter{}, usererror.BadRequest("Invalid value for the sort query parameter.")
	}

	return types.PublicKeyFilter{
		ListQueryFilter: ParseListQueryFilterFromRequest(r),
		Sort:            sort,
		Order:           ParseOrder(r),
	}, nil
}
