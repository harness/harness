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

package settings

import (
	"context"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/check"

	"github.com/gotidy/ptr"
)

// GeneralSettingsSpace represents the general space settings as exposed externally.
type GeneralSettingsSpace struct {
	DefaultBranch *string `json:"default_branch" yaml:"default_branch" description:"default branch name"`
}

func GetDefaultGeneralSettingsSpace() *GeneralSettingsSpace {
	return &GeneralSettingsSpace{
		DefaultBranch: ptr.String(DefaultBranch),
	}
}

func getGeneralSettingsMappingsSpace(s *GeneralSettingsSpace) []SettingHandler {
	return []SettingHandler{
		Mapping(DefaultBranchKey, s.DefaultBranch),
	}
}

func getGeneralSettingsMutationsSpace(s *GeneralSettingsSpace) ([]KeyValue, []Key) {
	kvs := make([]KeyValue, 0, 1)
	keysToDelete := make([]Key, 0, 1)

	if s.DefaultBranch != nil && *s.DefaultBranch != "" {
		kvs = append(kvs, KeyValue{
			Key:   DefaultBranchKey,
			Value: s.DefaultBranch,
		})
	}

	if s.DefaultBranch != nil && *s.DefaultBranch == "" {
		keysToDelete = append(keysToDelete, DefaultBranchKey)
	}

	return kvs, keysToDelete
}

// validateGeneralSettingsSpace validates values for general space settings updates.
func validateGeneralSettingsSpace(s *GeneralSettingsSpace) error {
	if s == nil || s.DefaultBranch == nil || *s.DefaultBranch == "" {
		return nil
	}

	if err := check.BranchName(*s.DefaultBranch); err != nil {
		return errors.InvalidArgumentf("invalid default branch name: %s", err)
	}

	return nil
}

// SpaceUpdateGeneralSettings updates general space settings and returns old and new full snapshots.
func SpaceUpdateGeneralSettings(
	ctx context.Context,
	service *Service,
	spaceID int64,
	in *GeneralSettingsSpace,
) (*GeneralSettingsSpace, *GeneralSettingsSpace, error) {
	err := validateGeneralSettingsSpace(in)
	if err != nil {
		return nil, nil, err
	}

	keyValues, deleteKeys := getGeneralSettingsMutationsSpace(in)

	old, out, err := SpaceUpdateWithDefaults(
		ctx,
		service,
		spaceID,
		GetDefaultGeneralSettingsSpace,
		getGeneralSettingsMappingsSpace,
		keyValues...,
	)
	if err != nil {
		return nil, nil, err
	}

	if len(deleteKeys) > 0 {
		err = service.SpaceDeleteMany(ctx, spaceID, deleteKeys...)
		if err != nil {
			return nil, nil, err
		}

		out, err = SpaceMapWithDefaults(
			ctx,
			service,
			spaceID,
			GetDefaultGeneralSettingsSpace,
			getGeneralSettingsMappingsSpace,
		)
		if err != nil {
			return nil, nil, err
		}
	}

	return old, out, nil
}
