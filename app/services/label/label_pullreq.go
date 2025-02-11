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

type AssignToPullReqOut struct {
	Label         *types.Label
	PullReqLabel  *types.PullReqLabel
	OldLabelValue *types.LabelValue
	NewLabelValue *types.LabelValue
	ActivityType  enum.PullReqLabelActivityType
}

type WithValue struct {
	Label *types.Label
	Value *types.LabelValue
}

func (out *AssignToPullReqOut) ToLabelPullReqAssignmentInfo() *types.LabelPullReqAssignmentInfo {
	var valueID *int64
	var value *string
	var valueColor *enum.LabelColor
	if out.NewLabelValue != nil {
		valueID = &out.NewLabelValue.ID
		value = &out.NewLabelValue.Value
		valueColor = &out.NewLabelValue.Color
	}

	assignmentInfo := &types.LabelPullReqAssignmentInfo{
		PullReqID:  out.PullReqLabel.PullReqID,
		LabelID:    out.Label.ID,
		LabelKey:   out.Label.Key,
		LabelColor: out.Label.Color,
		LabelScope: out.Label.Scope,
		ValueCount: out.Label.ValueCount,
		ValueID:    valueID,
		Value:      value,
		ValueColor: valueColor,
	}

	return assignmentInfo
}

func (s *Service) AssignToPullReq(
	ctx context.Context,
	principalID int64,
	pullreqID int64,
	repoID int64,
	repoParentID int64,
	in *types.PullReqLabelAssignInput,
) (*AssignToPullReqOut, error) {
	label, err := s.labelStore.FindByID(ctx, in.LabelID)
	if err != nil {
		return nil, fmt.Errorf("failed to find label by id: %w", err)
	}

	if err := s.checkPullreqLabelInScope(ctx, repoParentID, repoID, label); err != nil {
		return nil, err
	}

	oldPullreqLabel, err := s.pullReqLabelAssignmentStore.FindByLabelID(ctx, pullreqID, label.ID)
	if err != nil && !errors.Is(err, store.ErrResourceNotFound) {
		return nil, fmt.Errorf("failed to find label by id: %w", err)
	}

	// if the pullreq label did not have value
	if oldPullreqLabel != nil && oldPullreqLabel.ValueID == nil &&
		// and we don't assign it a new value
		in.Value == "" && in.ValueID == nil {
		return &AssignToPullReqOut{
			Label:         label,
			PullReqLabel:  oldPullreqLabel,
			OldLabelValue: nil,
			NewLabelValue: nil,
			ActivityType:  enum.LabelActivityNoop,
		}, nil
	}

	var oldLabelValue *types.LabelValue
	if oldPullreqLabel != nil && oldPullreqLabel.ValueID != nil {
		oldLabelValue, err = s.labelValueStore.FindByID(ctx, *oldPullreqLabel.ValueID)
		if err != nil {
			return nil, fmt.Errorf("failed to find label value by id: %w", err)
		}
	}

	// if the pullreq label had a value
	if oldLabelValue != nil {
		// and we reassign it the same value
		if in.ValueID != nil && oldLabelValue.ID == *in.ValueID {
			return &AssignToPullReqOut{
				Label:         label,
				PullReqLabel:  oldPullreqLabel,
				OldLabelValue: oldLabelValue,
				NewLabelValue: nil,
				ActivityType:  enum.LabelActivityNoop,
			}, nil
		}
		// and we reassign it the same value
		if in.Value != "" && oldLabelValue.Value == in.Value {
			return &AssignToPullReqOut{
				Label:         label,
				PullReqLabel:  oldPullreqLabel,
				OldLabelValue: oldLabelValue,
				NewLabelValue: nil,
				ActivityType:  enum.LabelActivityNoop,
			}, nil
		}
	}

	var newLabelValue *types.LabelValue
	if in.ValueID != nil {
		newLabelValue, err = s.labelValueStore.FindByID(ctx, *in.ValueID)
		if err != nil {
			return nil, fmt.Errorf("failed to find label value by id: %w", err)
		}
		if label.ID != newLabelValue.LabelID {
			return nil, errors.InvalidArgument("label value is not associated with label")
		}
	}

	newPullreqLabel := newPullReqLabel(pullreqID, principalID, in)
	if in.Value != "" {
		newLabelValue, err = s.getOrDefineValue(ctx, principalID, label, in.Value)
		if err != nil {
			return nil, err
		}
		newPullreqLabel.ValueID = &newLabelValue.ID
	}

	err = s.pullReqLabelAssignmentStore.Assign(ctx, newPullreqLabel)
	if err != nil {
		return nil, fmt.Errorf("failed to assign label to pullreq: %w", err)
	}

	activityType := enum.LabelActivityAssign
	if oldPullreqLabel != nil {
		activityType = enum.LabelActivityReassign
	}

	return &AssignToPullReqOut{
		Label:         label,
		PullReqLabel:  newPullreqLabel,
		OldLabelValue: oldLabelValue,
		NewLabelValue: newLabelValue,
		ActivityType:  activityType,
	}, nil
}

func (s *Service) PreparePullReqLabel(
	ctx context.Context,
	principalID int64,
	repoID int64,
	repoParentID int64,
	in *types.PullReqLabelAssignInput,
) (WithValue, error) {
	label, err := s.labelStore.FindByID(ctx, in.LabelID)
	if err != nil {
		return WithValue{}, fmt.Errorf("failed to find label by id: %w", err)
	}

	if err := s.checkPullreqLabelInScope(ctx, repoParentID, repoID, label); err != nil {
		return WithValue{}, err
	}

	var value *types.LabelValue
	if in.ValueID != nil {
		value, err = s.labelValueStore.FindByID(ctx, *in.ValueID)
		if err != nil {
			return WithValue{}, fmt.Errorf("failed to find label value by id: %w", err)
		}
		if label.ID != value.LabelID {
			return WithValue{}, errors.InvalidArgument("label value is not associated with label")
		}
	} else if in.Value != "" {
		value, err = s.getOrDefineValue(ctx, principalID, label, in.Value)
		if err != nil {
			return WithValue{}, err
		}
	}

	return WithValue{
		Label: label,
		Value: value,
	}, nil
}
func (s *Service) AssignToPullReqOnCreation(
	ctx context.Context,
	pullreqID int64,
	principalID int64,
	labelWithValue *WithValue,
	labelAssignInput *types.PullReqLabelAssignInput,
) (*AssignToPullReqOut, error) {
	pullReqLabel := newPullReqLabel(pullreqID, principalID, labelAssignInput)
	if labelWithValue.Value != nil {
		pullReqLabel.ValueID = &labelWithValue.Value.ID
	}

	err := s.pullReqLabelAssignmentStore.Assign(ctx, pullReqLabel)
	if err != nil {
		return nil, fmt.Errorf("failed to assign label to pullreq: %w", err)
	}

	return &AssignToPullReqOut{
		Label:         labelWithValue.Label,
		PullReqLabel:  pullReqLabel,
		NewLabelValue: labelWithValue.Value,
		ActivityType:  enum.LabelActivityAssign,
	}, nil
}

func (s *Service) getOrDefineValue(
	ctx context.Context,
	principalID int64,
	label *types.Label,
	value string,
) (*types.LabelValue, error) {
	if label.Type != enum.LabelTypeDynamic {
		return nil, errors.InvalidArgument("label doesn't allow new value assignment")
	}

	labelValue, err := s.labelValueStore.FindByLabelID(ctx, label.ID, value)
	if err == nil {
		return labelValue, nil
	}
	if !errors.Is(err, store.ErrResourceNotFound) {
		return nil, fmt.Errorf("failed to find label value: %w", err)
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
		return nil, fmt.Errorf("failed to create label value: %w", err)
	}
	label.ValueCount++

	return labelValue, nil
}

func (s *Service) UnassignFromPullReq(
	ctx context.Context, repoID, repoParentID, pullreqID, labelID int64,
) (*types.Label, *types.LabelValue, error) {
	label, err := s.labelStore.FindByID(ctx, labelID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find label by id: %w", err)
	}

	if err := s.checkPullreqLabelInScope(ctx, repoParentID, repoID, label); err != nil {
		return nil, nil, err
	}

	value, err := s.pullReqLabelAssignmentStore.FindValueByLabelID(ctx, pullreqID, labelID)
	if err != nil && !errors.Is(err, store.ErrResourceNotFound) {
		return nil, nil, fmt.Errorf("failed to find label value: %w", err)
	}

	return label, value, s.pullReqLabelAssignmentStore.Unassign(ctx, pullreqID, labelID)
}

func (s *Service) ListPullReqLabels(
	ctx context.Context,
	repo *types.Repository,
	spaceID int64,
	pullreqID int64,
	filter *types.AssignableLabelFilter,
) (*types.ScopesLabels, int64, error) {
	spaces, err := s.spaceStore.GetAncestors(ctx, spaceID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get space hierarchy: %w", err)
	}

	spaceIDs := make([]int64, len(spaces))
	for i, space := range spaces {
		spaceIDs[i] = space.ID
	}

	scopeLabelsMap := make(map[int64]*types.ScopeData)

	pullreqAssignments, err := s.pullReqLabelAssignmentStore.ListAssigned(ctx, pullreqID)
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

	total, err := s.labelStore.CountInScopes(ctx, repo.ID, spaceIDs, &types.LabelFilter{
		ListQueryFilter: types.ListQueryFilter{
			Query: filter.Query,
		},
	})
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

func (s *Service) Backfill(
	ctx context.Context,
	pullreq *types.PullReq,
) error {
	pullreqAssignments, err := s.pullReqLabelAssignmentStore.ListAssignedByPullreqIDs(
		ctx, []int64{pullreq.ID})
	if err != nil {
		return fmt.Errorf("failed to list labels assigned to pullreq: %w", err)
	}

	pullreq.Labels = pullreqAssignments[pullreq.ID]

	return nil
}

func (s *Service) BackfillMany(
	ctx context.Context,
	pullreqs []*types.PullReq,
) error {
	pullreqIDs := make([]int64, len(pullreqs))
	for i, pr := range pullreqs {
		pullreqIDs[i] = pr.ID
	}

	pullreqAssignments, err := s.pullReqLabelAssignmentStore.ListAssignedByPullreqIDs(
		ctx, pullreqIDs)
	if err != nil {
		return fmt.Errorf("failed to list labels assigned to pullreq: %w", err)
	}

	for _, pullreq := range pullreqs {
		pullreq.Labels = pullreqAssignments[pullreq.ID]
	}

	return nil
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
	in *types.PullReqLabelAssignInput,
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

func (s *Service) checkPullreqLabelInScope(
	ctx context.Context,
	repoParentID, repoID int64,
	label *types.Label,
) error {
	if label.RepoID != nil && *label.RepoID != repoID {
		return errors.InvalidArgument("label %d is not defined in current repo", label.ID)
	}

	if label.SpaceID != nil {
		spaceIDs, err := s.spaceStore.GetAncestorIDs(ctx, repoParentID)
		if err != nil {
			return fmt.Errorf("failed to get parent space ids: %w", err)
		}
		if ok := slices.Contains(spaceIDs, *label.SpaceID); !ok {
			return errors.InvalidArgument("label %d is not defined in current space tree path", label.ID)
		}
	}

	return nil
}

func (s *Service) backfillPullreqCount(
	ctx context.Context,
	labels []*types.Label,
) error {
	ids := make([]int64, len(labels))
	for i, label := range labels {
		ids[i] = label.ID
	}

	counts, err := s.pullReqLabelAssignmentStore.CountPullreqAssignments(ctx, ids)
	if err != nil {
		return fmt.Errorf("failed to count pullreq assignments: %w", err)
	}

	for _, label := range labels {
		label.PullreqCount = counts[label.ID]
	}

	return nil
}
