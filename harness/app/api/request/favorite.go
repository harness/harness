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
	"github.com/harness/gitness/types/enum"
)

const (
	PathParamResourceID = "resource_id"

	QueryParamResourceType = "resource_type"
)

// GetResourceIDFromPath extracts the resource id from the url path.
func GetResourceIDFromPath(r *http.Request) (int64, error) {
	return PathParamAsPositiveInt64(r, PathParamResourceID)
}

// ParseResourceType extracts the resource type from the url query param.
func ParseResourceType(r *http.Request) (enum.ResourceType, error) {
	resourceType, ok := enum.ResourceType(r.URL.Query().Get(QueryParamResourceType)).Sanitize()
	if !ok {
		return "", usererror.BadRequest("Invalid resource type")
	}

	return resourceType, nil
}
