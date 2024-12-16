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

package space

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// GetSpaceCheckAuth checks whether the user has the requested permission on the provided space and returns the space.
func GetSpaceCheckAuth(
	ctx context.Context,
	spaceStore store.SpaceStore,
	authorizer authz.Authorizer,
	session *auth.Session,
	spaceRef string,
	permission enum.Permission,
) (*types.Space, error) {
	space, err := spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("space not found: %w", err)
	}

	err = apiauth.CheckSpace(ctx, authorizer, session, space, permission)
	if err != nil {
		return nil, fmt.Errorf("auth check failed: %w", err)
	}

	return space, nil
}

func GetSpaceOutput(
	ctx context.Context,
	publicAccess publicaccess.Service,
	space *types.Space,
) (*SpaceOutput, error) {
	isPublic, err := publicAccess.Get(ctx, enum.PublicResourceTypeSpace, space.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource public access mode: %w", err)
	}

	return &SpaceOutput{
		Space:    *space,
		IsPublic: isPublic,
	}, nil
}
