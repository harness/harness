// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

import "sort"

const (
	id            = "id"
	uid           = "uid"
	path          = "path"
	name          = "name"
	number        = "number"
	created       = "created"
	createdAt     = "created_at"
	createdBy     = "created_by"
	updated       = "updated"
	updatedAt     = "updated_at"
	updatedBy     = "updated_by"
	date          = "date"
	defaultString = "default"
	undefined     = "undefined"
	system        = "system"
	comment       = "comment"
	code          = "code"
)

func existsInSortedSlice(strs []string, s string) bool {
	idx := sort.SearchStrings(strs, s)
	return idx >= 0 && idx < len(strs) && strs[idx] == s
}

func enumToStringSlice[T ~string](vals []T) []string {
	res := make([]string, len(vals))
	for i := range vals {
		res[i] = string(vals[i])
	}
	return res
}
