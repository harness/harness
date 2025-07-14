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

package auth

import (
	"context"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// CheckInfraProvider checks if a gitspace specific permission is granted for the current auth session
// in the scope of its parent.
// Returns nil if the permission is granted, otherwise returns an error.
// NotAuthenticated, NotAuthorized, or any underlying error.
func CheckInfraProvider(
	ctx context.Context,
	authorizer authz.Authorizer,
	session *auth.Session,
	parentPath,
	identifier string,
	permission enum.Permission,
) error {
	scope := &types.Scope{SpacePath: parentPath}
	resource := &types.Resource{
		Type:       enum.ResourceTypeInfraProvider,
		Identifier: identifier,
	}

	return Check(ctx, authorizer, session, scope, resource, permission)
}
