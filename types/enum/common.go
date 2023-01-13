// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

import (
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

func Sanitize[E constraints.Ordered](element E, all func() ([]E, E)) (E, bool) {
	allValues, defValue := all()
	var empty E
	if element == empty && defValue != empty {
		return defValue, true
	}
	idx, exists := slices.BinarySearch(allValues, element)
	if exists {
		return allValues[idx], true
	}
	return defValue, false
}

const (
	id            = "id"
	uid           = "uid"
	path          = "path"
	name          = "name"
	email         = "email"
	admin         = "admin"
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
	asc           = "asc"
	ascending     = "ascending"
	desc          = "desc"
	descending    = "descending"
)

func toInterfaceSlice[T interface{}](vals []T) []interface{} {
	res := make([]interface{}, len(vals))
	for i := range vals {
		res[i] = vals[i]
	}
	return res
}

func sortEnum[T constraints.Ordered](slice []T) []T {
	slices.Sort(slice)
	return slice
}
