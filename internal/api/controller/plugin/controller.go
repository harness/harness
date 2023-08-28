// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package plugin

import (
	"github.com/harness/gitness/internal/store"

	"github.com/jmoiron/sqlx"
)

type Controller struct {
	db          *sqlx.DB
	pluginStore store.PluginStore
}

func NewController(
	db *sqlx.DB,
	pluginStore store.PluginStore,
) *Controller {
	return &Controller{
		db:          db,
		pluginStore: pluginStore,
	}
}
