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

package autolink

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/types"
)

type Scope struct {
	SpaceID *int64
	RepoID  *int64
}

func checkScope(autolink *types.AutoLink, scope Scope) error {
	if scope.RepoID != nil {
		if autolink.RepoID != nil && *autolink.RepoID == *scope.RepoID {
			return nil
		}
		return usererror.ErrNotFound
	}

	if scope.SpaceID != nil {
		if autolink.SpaceID != nil && *autolink.SpaceID == *scope.SpaceID {
			return nil
		}
		return usererror.ErrNotFound
	}

	return usererror.ErrNotFound
}

func (s *Service) findAndCheck(
	ctx context.Context,
	scope Scope,
	id int64,
) (*types.AutoLink, error) {
	autolink, err := s.autoLinkStore.Find(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get autolink: %w", err)
	}

	if err := checkScope(autolink, scope); err != nil {
		return nil, err
	}

	return autolink, nil
}
