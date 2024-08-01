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
	"sort"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
)

func (s *Service) Define(
	ctx context.Context,
	principalID int64,
	spaceID, repoID *int64,
	in *types.DefineLabelInput,
) (*types.Label, error) {
	var scope int64
	if spaceID != nil {
		spaceIDs, err := s.spaceStore.GetAncestorIDs(ctx, *spaceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get space ids hierarchy: %w", err)
		}
		scope = int64(len(spaceIDs))
	}

	label := newLabel(principalID, spaceID, repoID, scope, in)

	if err := s.labelStore.Define(ctx, label); err != nil {
		return nil, err
	}

	return label, nil
}

func (s *Service) Update(
	ctx context.Context,
	principalID int64,
	spaceID, repoID *int64,
	key string,
	in *types.UpdateLabelInput,
) (*types.Label, error) {
	label, err := s.labelStore.Find(ctx, spaceID, repoID, key)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo label: %w", err)
	}

	return s.update(ctx, principalID, label, in)
}

func (s *Service) update(
	ctx context.Context,
	principalID int64,
	label *types.Label,
	in *types.UpdateLabelInput,
) (*types.Label, error) {
	label, hasChanges := applyChanges(principalID, label, in)
	if !hasChanges {
		return label, nil
	}

	err := s.labelStore.Update(ctx, label)
	if err != nil {
		return nil, fmt.Errorf("failed to update label: %w", err)
	}

	return label, nil
}

//nolint:gocognit
func (s *Service) Save(
	ctx context.Context,
	principalID int64,
	spaceID, repoID *int64,
	in *types.SaveInput,
) (*types.LabelWithValues, error) {
	var label *types.Label
	var valuesToReturn []*types.LabelValue
	var err error

	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		label, err = s.labelStore.FindByID(ctx, in.Label.ID)
		if err != nil {
			if !errors.Is(err, store.ErrResourceNotFound) {
				return err
			}
			label, err = s.Define(ctx, principalID, spaceID, repoID, &in.Label.DefineLabelInput)
			if err != nil {
				return err
			}
		} else {
			label, err = s.update(ctx, principalID, label, &types.UpdateLabelInput{
				Key:         &in.Label.Key,
				Type:        &in.Label.Type,
				Description: &in.Label.Description,
				Color:       &in.Label.Color,
			})
			if err != nil {
				return err
			}
		}

		existingValues, err := s.labelValueStore.List(ctx, label.ID, &types.ListQueryFilter{})
		if err != nil {
			return err
		}
		existingValuesMap := make(map[int64]*types.LabelValue, len(existingValues))
		for _, value := range existingValues {
			existingValuesMap[value.ID] = value
		}

		var valuesToCreate []*types.SaveLabelValueInput
		valuesToUpdate := make(map[int64]*types.SaveLabelValueInput)
		var valuesToDelete []string

		for _, value := range in.Values {
			if _, ok := existingValuesMap[value.ID]; ok {
				valuesToUpdate[value.ID] = value
			} else {
				valuesToCreate = append(valuesToCreate, value)
			}
		}

		for _, value := range existingValues {
			if _, ok := valuesToUpdate[value.ID]; !ok {
				valuesToDelete = append(valuesToDelete, value.Value)
			}
		}

		valuesToReturn = make([]*types.LabelValue, len(valuesToCreate)+len(valuesToUpdate))

		for i, value := range valuesToCreate {
			valuesToReturn[i] = newLabelValue(principalID, label.ID, &value.DefineValueInput)
			if err = s.labelValueStore.Define(ctx, valuesToReturn[i]); err != nil {
				return err
			}
		}

		i := len(valuesToCreate)
		for _, value := range valuesToUpdate {
			if valuesToReturn[i], err = s.updateValue(ctx, principalID, existingValuesMap[value.ID], &types.UpdateValueInput{
				Value: &value.Value,
				Color: &value.Color,
			}); err != nil {
				return err
			}
			i++
		}

		if err = s.labelValueStore.DeleteMany(ctx, label.ID, valuesToDelete); err != nil {
			return err
		}

		if label.ValueCount, err = s.labelStore.IncrementValueCount(
			ctx, label.ID, len(valuesToCreate)-len(valuesToDelete)); err != nil {
			return err
		}

		sort.Slice(valuesToReturn, func(i, j int) bool {
			return valuesToReturn[i].Value < valuesToReturn[j].Value
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save label: %w", err)
	}

	return &types.LabelWithValues{
		Label:  *label,
		Values: valuesToReturn,
	}, nil
}

func (s *Service) Find(
	ctx context.Context,
	spaceID, repoID *int64,
	key string,
) (*types.Label, error) {
	return s.labelStore.Find(ctx, spaceID, repoID, key)
}

func (s *Service) FindByID(ctx context.Context, labelID int64) (*types.Label, error) {
	return s.labelStore.FindByID(ctx, labelID)
}

func (s *Service) List(
	ctx context.Context,
	spaceID, repoID *int64,
	filter *types.LabelFilter,
) ([]*types.Label, int64, error) {
	if filter.Inherited {
		return s.listInScopes(ctx, spaceID, repoID, filter)
	}

	return s.list(ctx, spaceID, repoID, filter)
}

func (s *Service) list(
	ctx context.Context,
	spaceID, repoID *int64,
	filter *types.LabelFilter,
) ([]*types.Label, int64, error) {
	if repoID != nil {
		total, err := s.labelStore.CountInRepo(ctx, *repoID, filter)
		if err != nil {
			return nil, 0, err
		}

		labels, err := s.labelStore.List(ctx, nil, repoID, filter)
		if err != nil {
			return nil, 0, err
		}
		return labels, total, nil
	}

	count, err := s.labelStore.CountInSpace(ctx, *spaceID, filter)
	if err != nil {
		return nil, 0, err
	}
	labels, err := s.labelStore.List(ctx, spaceID, nil, filter)
	if err != nil {
		return nil, 0, err
	}
	return labels, count, nil
}

func (s *Service) listInScopes(
	ctx context.Context,
	spaceID, repoID *int64,
	filter *types.LabelFilter,
) ([]*types.Label, int64, error) {
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

	total, err := s.labelStore.CountInScopes(ctx, repoIDVal, spaceIDs, filter)
	if err != nil {
		return nil, 0, err
	}

	labels, err := s.labelStore.ListInScopes(ctx, repoIDVal, spaceIDs, filter)
	if err != nil {
		return nil, 0, err
	}

	return labels, total, nil
}

func (s *Service) Delete(
	ctx context.Context,
	spaceID, repoID *int64,
	key string,
) error {
	return s.labelStore.Delete(ctx, spaceID, repoID, key)
}

func newLabel(
	principalID int64,
	spaceID, repoID *int64,
	scope int64,
	in *types.DefineLabelInput,
) *types.Label {
	now := time.Now().UnixMilli()
	return &types.Label{
		RepoID:      repoID,
		SpaceID:     spaceID,
		Scope:       scope,
		Key:         in.Key,
		Type:        in.Type,
		Description: in.Description,
		Color:       in.Color,
		Created:     now,
		Updated:     now,
		CreatedBy:   principalID,
		UpdatedBy:   principalID,
	}
}

func applyChanges(principalID int64, label *types.Label, in *types.UpdateLabelInput) (*types.Label, bool) {
	hasChanges := false

	if label.UpdatedBy != principalID {
		hasChanges = true
		label.UpdatedBy = principalID
	}
	if in.Key != nil && label.Key != *in.Key {
		hasChanges = true
		label.Key = *in.Key
	}
	if in.Description != nil && label.Description != *in.Description {
		hasChanges = true
		label.Description = *in.Description
	}
	if in.Color != nil && label.Color != *in.Color {
		hasChanges = true
		label.Color = *in.Color
	}
	if in.Type != nil && label.Type != *in.Type {
		hasChanges = true
		label.Type = *in.Type
	}

	if hasChanges {
		label.Updated = time.Now().UnixMilli()
	}

	return label, hasChanges
}
