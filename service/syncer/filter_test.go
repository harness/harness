// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package syncer

import (
	"testing"

	"github.com/drone/drone/core"
)

func TestNamespaceFilter(t *testing.T) {
	tests := []struct {
		namespace  string
		namespaces []string
		match      bool
	}{
		{
			namespace:  "octocat",
			namespaces: []string{"octocat"},
			match:      true,
		},
		{
			namespace:  "OCTocat",
			namespaces: []string{"octOCAT"},
			match:      true,
		},
		{
			namespace:  "spaceghost",
			namespaces: []string{"octocat"},
			match:      false,
		},
		{
			namespace:  "spaceghost",
			namespaces: []string{},
			match:      true, // no-op filter
		},
	}
	for _, test := range tests {
		r := &core.Repository{Namespace: test.namespace}
		f := NamespaceFilter(test.namespaces)
		if got, want := f(r), test.match; got != want {
			t.Errorf("Want match %v for namespace %q and namespaces %v", want, test.namespace, test.namespaces)
		}
	}
}
