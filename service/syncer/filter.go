// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package syncer

import (
	"strings"

	"github.com/drone/drone/core"
)

// FilterFunc can be used to filter which repositories are
// synchronized with the local datastore.
type FilterFunc func(*core.Repository) bool

// NamespaceFilter is a filter function that returns true
// if the repository namespace matches a provided namespace
// in the list.
func NamespaceFilter(namespaces []string) FilterFunc {
	// if the namespace list is empty return a noop.
	if len(namespaces) == 0 {
		return noopFilter
	}
	return func(r *core.Repository) bool {
		for _, namespace := range namespaces {
			if strings.EqualFold(namespace, r.Namespace) {
				return true
			}
		}
		return false
	}
}

// noopFilter is a filter function that always returns true.
func noopFilter(*core.Repository) bool {
	return true
}
