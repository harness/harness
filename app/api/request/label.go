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
)

const (
	PathParamLabelKey   = "label_key"
	PathParamLabelValue = "label_value"
	PathParamLabelID    = "label_id"
)

func GetLabelKeyFromPath(r *http.Request) (string, error) {
	return EncodedPathParamOrError(r, PathParamLabelKey)
}

func GetLabelValueFromPath(r *http.Request) (string, error) {
	return EncodedPathParamOrError(r, PathParamLabelValue)
}

func GetLabelIDFromPath(r *http.Request) (int64, error) {
	return PathParamAsPositiveInt64(r, PathParamLabelID)
}

// ParseLabelFilter extracts the label filter from the url.
func ParseLabelFilter(r *http.Request) (*types.LabelFilter, error) {
	// inherited is used to list labels from parent scopes
	inherited, err := ParseInheritedFromQuery(r)
	if err != nil {
		return nil, err
	}

	return &types.LabelFilter{
		Inherited:       inherited,
		ListQueryFilter: ParseListQueryFilterFromRequest(r),
	}, nil
}

// ParseAssignableLabelFilter extracts the assignable label filter from the url.
func ParseAssignableLabelFilter(r *http.Request) (*types.AssignableLabelFilter, error) {
	// assignable is used to list all labels assignable to pullreq
	assignable, err := ParseAssignableFromQuery(r)
	if err != nil {
		return nil, err
	}

	return &types.AssignableLabelFilter{
		Assignable:      assignable,
		ListQueryFilter: ParseListQueryFilterFromRequest(r),
	}, nil
}
