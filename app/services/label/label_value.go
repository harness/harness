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
	"time"

	"github.com/harness/gitness/types"
)

func (s *Service) DefineValue(
	ctx context.Context,
	principalID int64,
	labelID int64,
	in *types.DefineValueInput,
) (*types.LabelValue, error) {
	labelValue := newLabelValue(principalID, labelID, in)

	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.labelValueStore.Define(ctx, labelValue); err != nil {
			return err
		}
		if _, err := s.labelStore.IncrementValueCount(ctx, labelID, 1); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return labelValue, nil
}

func applyValueChanges(
	principalID int64,
	value *types.LabelValue,
	in *types.UpdateValueInput,
) (*types.LabelValue, bool) {
	hasChanges := false

	if value.UpdatedBy != principalID {
		hasChanges = true
		value.UpdatedBy = principalID
	}

	if in.Value != nil && value.Value != *in.Value {
		hasChanges = true
		value.Value = *in.Value
	}
	if in.Color != nil && value.Color != *in.Color {
		hasChanges = true
		value.Color = *in.Color
	}

	if hasChanges {
		value.Updated = time.Now().UnixMilli()
	}

	return value, hasChanges
}

func (s *Service) UpdateValue(
	ctx context.Context,
	principalID int64,
	labelID int64,
	value string,
	in *types.UpdateValueInput,
) (*types.LabelValue, error) {
	labelValue, err := s.labelValueStore.FindByLabelID(ctx, labelID, value)
	if err != nil {
		return nil, fmt.Errorf("failed to find label value: %w", err)
	}

	return s.updateValue(ctx, principalID, labelValue, in)
}

func (s *Service) updateValue(
	ctx context.Context,
	principalID int64,
	labelValue *types.LabelValue,
	in *types.UpdateValueInput,
) (*types.LabelValue, error) {
	labelValue, hasChanges := applyValueChanges(
		principalID, labelValue, in)
	if !hasChanges {
		return labelValue, nil
	}

	if err := s.labelValueStore.Update(ctx, labelValue); err != nil {
		return nil, fmt.Errorf("failed to update label value: %w", err)
	}

	return labelValue, nil
}

func (s *Service) ListValues(
	ctx context.Context,
	spaceID, repoID *int64,
	labelKey string,
	filter *types.ListQueryFilter,
) ([]*types.LabelValue, error) {
	label, err := s.labelStore.Find(ctx, spaceID, repoID, labelKey)
	if err != nil {
		return nil, err
	}

	return s.labelValueStore.List(ctx, label.ID, filter)
}

func (s *Service) DeleteValue(
	ctx context.Context,
	spaceID, repoID *int64,
	labelKey string,
	value string,
) error {
	label, err := s.labelStore.Find(ctx, spaceID, repoID, labelKey)
	if err != nil {
		return err
	}

	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.labelValueStore.Delete(ctx, label.ID, value); err != nil {
			return err
		}
		if _, err := s.labelStore.IncrementValueCount(ctx, label.ID, -1); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func newLabelValue(
	principalID int64,
	labelID int64,
	in *types.DefineValueInput,
) *types.LabelValue {
	now := time.Now().UnixMilli()
	return &types.LabelValue{
		LabelID:   labelID,
		Value:     in.Value,
		Color:     in.Color,
		Created:   now,
		Updated:   now,
		CreatedBy: principalID,
		UpdatedBy: principalID,
	}
}
