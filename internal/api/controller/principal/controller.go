// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package principal

import (
	"github.com/harness/gitness/internal/store"
)

type Controller struct {
	principalStore store.PrincipalStore
}

func NewController(principalStore store.PrincipalStore) *Controller {
	return &Controller{
		principalStore: principalStore,
	}
}
