// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package principal

import (
	"github.com/harness/gitness/internal/store"
)

type controller struct {
	principalStore store.PrincipalStore
}

func newController(principalStore store.PrincipalStore) *controller {
	return &controller{
		principalStore: principalStore,
	}
}
