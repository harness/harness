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

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (s *Service) List(
	ctx context.Context,
	spaceID, repoID *int64,
	filter *types.AutoLinkFilter,
) ([]*types.AutoLink, int64, error) {
	var autoLinks []*types.AutoLink
	var count int64
	var err error

	if filter.Inherited {
		autoLinks, count, err = s.listInScopes(ctx, spaceID, repoID, filter)
	} else {
		autoLinks, count, err = s.list(ctx, spaceID, repoID, filter)
	}
	if err != nil {
		return nil, 0, err
	}

	for _, autoLink := range autoLinks {
		//nolint:exhaustive
		switch autoLink.Type {
		case enum.AutoLinkTypePrefixWithNumValue:
			autoLink.Pattern = "^" + autoLink.Pattern + "(\\d+)$"
			autoLink.Type = enum.AutoLinkTypeRegex
		case enum.AutoLinkTypePrefixWithAlphanumericValue:
			autoLink.Pattern = "^" + autoLink.Pattern + "([\\w-]+)$"
			autoLink.Type = enum.AutoLinkTypeRegex
		}
	}

	return autoLinks, count, nil
}

func (s *Service) list(
	ctx context.Context,
	spaceID, repoID *int64,
	filter *types.AutoLinkFilter,
) ([]*types.AutoLink, int64, error) {
	count, err := s.autoLinkStore.Count(ctx, spaceID, repoID, filter)
	if err != nil {
		return nil, 0, err
	}

	autoLinks, err := s.autoLinkStore.List(ctx, spaceID, repoID, filter)
	if err != nil {
		return nil, 0, err
	}

	return autoLinks, count, nil
}

func (s *Service) listInScopes(
	ctx context.Context,
	spaceID, repoID *int64,
	filter *types.AutoLinkFilter,
) ([]*types.AutoLink, int64, error) {
	var spaceIDs []int64
	var repoIDVal int64
	var err error

	if repoID != nil {
		spaceIDs, err = s.spaceStore.GetAncestorIDs(ctx, *spaceID)
		if err != nil {
			return nil, 0, err
		}
		repoIDVal = *repoID
	} else {
		spaceIDs, err = s.spaceStore.GetAncestorIDs(ctx, *spaceID)
		if err != nil {
			return nil, 0, err
		}
	}

	count, err := s.autoLinkStore.CountInScopes(ctx, repoIDVal, spaceIDs, filter)
	if err != nil {
		return nil, 0, err
	}

	autoLinks, err := s.autoLinkStore.ListInScopes(ctx, repoIDVal, spaceIDs, filter)
	if err != nil {
		return nil, 0, err
	}

	return autoLinks, count, nil
}
