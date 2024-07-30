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

package label

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"golang.org/x/exp/maps"
)

func (s *Service) AssignToPullReq(
	ctx context.Context,
	principalID int64,
	pullreqID int64,
	repoID int64,
	repoParentID int64,
	in *types.PullReqCreateInput,
) (*types.PullReqLabel, error) {
	label, err := s.labelStore.FindByID(ctx, in.LabelID)
	if err != nil {
		return nil, fmt.Errorf("failed to find label by id: %w", err)
	}

	if label.SpaceID != nil {
		spaceIDs, err := s.spaceStore.GetAncestorIDs(ctx, repoParentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent space ids: %w", err)
		}
		if ok := slices.Contains(spaceIDs, *label.SpaceID); !ok {
			return nil, errors.NotFound(
				"label %d is not defined in current space tree path", label.ID)
		}
	} else if label.RepoID != nil && *label.RepoID != repoID {
		return nil, errors.InvalidArgument(
			"label %d is not defined in current repo", label.ID)
	}

	pullreqLabel := newPullReqLabel(pullreqID, principalID, in)

	if in.ValueID != nil {
		labelValue, err := s.labelValueStore.FindByID(ctx, *in.ValueID)
		if err != nil {
			return nil, fmt.Errorf("failed to find label value by id: %w", err)
		}
		if label.ID != labelValue.LabelID {
			return nil, errors.InvalidArgument("label value is not associated with label")
		}
	}

	if in.Value != "" {
		valueID, err := s.getOrDefineValue(ctx, principalID, label, in.Value)
		if err != nil {
			return nil, err
		}
		pullreqLabel.ValueID = &valueID
	}

	err = s.pullReqLabelAssignmentStore.Assign(ctx, pullreqLabel)
	if err != nil {
		return nil, fmt.Errorf("failed to assign label to pullreq: %w", err)
	}

	return pullreqLabel, nil
}

func (s *Service) getOrDefineValue(
	ctx context.Context,
	principalID int64,
	label *types.Label,
	value string,
) (int64, error) {
	if label.Type != enum.LabelTypeDynamic {
		return 0, errors.InvalidArgument("label doesn't allow new value assignment")
	}

	labelValue, err := s.labelValueStore.FindByLabelID(ctx, label.ID, value)
	if err == nil {
		return labelValue.ID, nil
	}
	if !errors.Is(err, store.ErrResourceNotFound) {
		return 0, fmt.Errorf("failed to find label value: %w", err)
	}

	labelValue, err = s.DefineValue(
		ctx,
		principalID,
		label.ID,
		&types.DefineValueInput{
			Value: value,
			Color: label.Color,
		},
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create label value: %w", err)
	}

	return labelValue.ID, nil
}

func (s *Service) UnassignFromPullReq(
	ctx context.Context, repoID, repoParentID, pullreqID, labelID int64,
) error {
	label, err := s.labelStore.FindByID(ctx, labelID)
	if err != nil {
		return fmt.Errorf("failed to find label by id: %w", err)
	}

	if label.RepoID != nil && *label.RepoID != repoID {
		return errors.InvalidArgument(
			"label %d is not defined in current repo", label.ID)
	} else if label.SpaceID != nil {
		spaceIDs, err := s.spaceStore.GetAncestorIDs(ctx, repoParentID)
		if err != nil {
			return fmt.Errorf("failed to get parent space ids: %w", err)
		}
		if ok := slices.Contains(spaceIDs, *label.SpaceID); !ok {
			return errors.NotFound(
				"label %d is not defined in current space tree path", label.ID)
		}
	}

	return s.pullReqLabelAssignmentStore.Unassign(ctx, pullreqID, labelID)
}

func (s *Service) ListPullReqLabels(
	ctx context.Context,
	repo *types.Repository,
	spaceID int64,
	pullreqID int64,
	filter *types.AssignableLabelFilter,
) (*types.ScopesLabels, int64, error) {
	spaces, err := s.spaceStore.GetHierarchy(ctx, spaceID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get space hierarchy: %w", err)
	}

	spaceIDs := make([]int64, len(spaces))
	for i, space := range spaces {
		spaceIDs[i] = space.ID
	}

	scopeLabelsMap := make(map[int64]*types.ScopeData)

	pullreqAssignments, err := s.pullReqLabelAssignmentStore.ListAssigned(
		ctx, pullreqID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list labels assigned to pullreq: %w", err)
	}

	if !filter.Assignable {
		sortedAssignments := maps.Values(pullreqAssignments)
		sort.Slice(sortedAssignments, func(i, j int) bool {
			if sortedAssignments[i].Key != sortedAssignments[j].Key {
				return sortedAssignments[i].Key < sortedAssignments[j].Key
			}
			return sortedAssignments[i].Scope < sortedAssignments[j].Scope
		})

		populateScopeLabelsMap(sortedAssignments, scopeLabelsMap, repo, spaces)
		return createScopeLabels(sortedAssignments, scopeLabelsMap), 0, nil
	}

	total, err := s.labelStore.CountInScopes(ctx, repo.ID, spaceIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count labels in scopes: %w", err)
	}

	labelInfos, err := s.labelStore.ListInfosInScopes(ctx, repo.ID, spaceIDs, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list repo and spaces label infos: %w", err)
	}

	labelIDs := make([]int64, len(labelInfos))
	for i, labelInfo := range labelInfos {
		labelIDs[i] = labelInfo.ID
	}

	valueInfos, err := s.labelValueStore.ListInfosByLabelIDs(ctx, labelIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list label value infos by label ids: %w", err)
	}

	allAssignments := make([]*types.LabelAssignment, len(labelInfos))
	for i, labelInfo := range labelInfos {
		assignment, ok := pullreqAssignments[labelInfo.ID]
		if !ok {
			assignment = &types.LabelAssignment{
				LabelInfo: *labelInfo,
			}
		}
		assignment.LabelInfo.Assigned = &ok
		allAssignments[i] = assignment
		allAssignments[i].Values = valueInfos[labelInfo.ID]
	}

	populateScopeLabelsMap(allAssignments, scopeLabelsMap, repo, spaces)
	return createScopeLabels(allAssignments, scopeLabelsMap), total, nil
}

func populateScopeLabelsMap(
	assignments []*types.LabelAssignment,
	scopeLabelsMap map[int64]*types.ScopeData,
	repo *types.Repository,
	spaces []*types.Space,
) {
	for _, assignment := range assignments {
		_, ok := scopeLabelsMap[assignment.Scope]
		if ok {
			continue
		}
		scopeLabelsMap[assignment.Scope] = &types.ScopeData{Scope: assignment.Scope}
		if assignment.Scope == 0 {
			scopeLabelsMap[assignment.Scope].Repo = repo
		} else {
			for _, space := range spaces {
				if space.ID == *assignment.SpaceID {
					scopeLabelsMap[assignment.Scope].Space = space
				}
			}
		}
	}
}

func createScopeLabels(
	assignments []*types.LabelAssignment,
	scopeLabelsMap map[int64]*types.ScopeData,
) *types.ScopesLabels {
	scopeData := make([]*types.ScopeData, len(scopeLabelsMap))
	for i, scopeLabel := range maps.Values(scopeLabelsMap) {
		scopeData[i] = scopeLabel
	}

	sort.Slice(scopeData, func(i, j int) bool {
		return scopeData[i].Scope < scopeData[j].Scope
	})

	return &types.ScopesLabels{
		LabelData: assignments,
		ScopeData: scopeData,
	}
}

func newPullReqLabel(
	pullreqID int64,
	principalID int64,
	in *types.PullReqCreateInput,
) *types.PullReqLabel {
	now := time.Now().UnixMilli()
	return &types.PullReqLabel{
		PullReqID: pullreqID,
		LabelID:   in.LabelID,
		ValueID:   in.ValueID,
		Created:   now,
		Updated:   now,
		CreatedBy: principalID,
		UpdatedBy: principalID,
	}
}
