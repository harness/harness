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

package api

func mapRawRef(
	raw map[string]string,
) (map[GitReferenceField]string, error) {
	res := make(map[GitReferenceField]string, len(raw))
	for k, v := range raw {
		gitRefField, err := ParseGitReferenceField(k)
		if err != nil {
			return nil, err
		}
		res[gitRefField] = v
	}

	return res, nil
}

func mapToReferenceSortingArgument(
	s GitReferenceField,
	o SortOrder,
) string {
	sortBy := string(GitReferenceFieldRefName)
	desc := o == SortOrderDesc

	if s == GitReferenceFieldCreatorDate {
		sortBy = string(GitReferenceFieldCreatorDate)
		if o == SortOrderDefault {
			desc = true
		}
	}

	if desc {
		return "-" + sortBy
	}

	return sortBy
}
