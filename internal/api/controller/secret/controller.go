// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package secret

import (
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/check"

	"github.com/jmoiron/sqlx"
)

type Controller struct {
	db          *sqlx.DB
	uidCheck    check.PathUID
	encrypter   encrypt.Encrypter
	secretStore store.SecretStore
	authorizer  authz.Authorizer
	spaceStore  store.SpaceStore
}

func NewController(
	db *sqlx.DB,
	uidCheck check.PathUID,
	authorizer authz.Authorizer,
	encrypter encrypt.Encrypter,
	secretStore store.SecretStore,
	spaceStore store.SpaceStore,
) *Controller {
	return &Controller{
		db:          db,
		uidCheck:    uidCheck,
		encrypter:   encrypter,
		secretStore: secretStore,
		authorizer:  authorizer,
		spaceStore:  spaceStore,
	}
}
