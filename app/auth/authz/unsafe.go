// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package authz

import (
	"context"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

var _ Authorizer = (*UnsafeAuthorizer)(nil)

/*
 * An unsafe authorizer that gives permits any action and simply logs the permission request.
 */
type UnsafeAuthorizer struct{}

func NewUnsafeAuthorizer() *UnsafeAuthorizer {
	return &UnsafeAuthorizer{}
}

func (a *UnsafeAuthorizer) Check(ctx context.Context, session *auth.Session,
	scope *types.Scope, resource *types.Resource, permission enum.Permission) (bool, error) {
	log.Ctx(ctx).Info().Msgf(
		"[Authz] %s with id '%d' requests %s for %s '%s' in scope %#v with metadata %#v",
		session.Principal.Type,
		session.Principal.ID,
		permission,
		resource.Type,
		resource.Identifier,
		scope,
		session.Metadata,
	)

	return true, nil
}
func (a *UnsafeAuthorizer) CheckAll(ctx context.Context, session *auth.Session,
	permissionChecks ...types.PermissionCheck) (bool, error) {
	for i := range permissionChecks {
		p := permissionChecks[i]
		if _, err := a.Check(ctx, session, &p.Scope, &p.Resource, p.Permission); err != nil {
			return false, err
		}
	}

	return true, nil
}
