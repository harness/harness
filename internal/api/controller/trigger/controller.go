// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package trigger

import (
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"

	"github.com/jmoiron/sqlx"
)

type Controller struct {
	db            *sqlx.DB
	authorizer    authz.Authorizer
	triggerStore  store.TriggerStore
	pipelineStore store.PipelineStore
	spaceStore    store.SpaceStore
}

func NewController(
	db *sqlx.DB,
	authorizer authz.Authorizer,
	triggerStore store.TriggerStore,
	pipelineStore store.PipelineStore,
	spaceStore store.SpaceStore,
) *Controller {
	return &Controller{
		db:            db,
		authorizer:    authorizer,
		triggerStore:  triggerStore,
		pipelineStore: pipelineStore,
		spaceStore:    spaceStore,
	}
}
