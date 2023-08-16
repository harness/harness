// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package secret

import (
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/check"

	"github.com/google/wire"
	"github.com/jmoiron/sqlx"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideController,
)

func ProvideController(db *sqlx.DB,
	uidCheck check.PathUID,
	pathStore store.PathStore,
	encrypter encrypt.Encrypter,
	secretStore store.SecretStore,
	authorizer authz.Authorizer,
	spaceStore store.SpaceStore,
) *Controller {
	return NewController(db, uidCheck, authorizer, pathStore, encrypter, secretStore, spaceStore)
}
