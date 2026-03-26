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
	"fmt"

	"github.com/harness/gitness/types/enum"
)

// SpaceSetMany sets the value of the settings with the given keys for the given space.
func (s *Service) SpaceSetMany(
	ctx context.Context,
	spaceID int64,
	keyValues ...KeyValue,
) error {
	return s.SetMany(
		ctx,
		enum.SettingsScopeSpace,
		spaceID,
		keyValues...,
	)
}

// SpaceDeleteMany deletes settings with the given keys for the given space.
func (s *Service) SpaceDeleteMany(
	ctx context.Context,
	spaceID int64,
	keys ...Key,
) error {
	return s.DeleteMany(
		ctx,
		enum.SettingsScopeSpace,
		spaceID,
		keys...,
	)
}

// SpaceGet returns the value of the setting with the given key for the given space.
func (s *Service) SpaceGet(
	ctx context.Context,
	spaceID int64,
	key Key,
	out any,
) (bool, error) {
	return s.Get(
		ctx,
		enum.SettingsScopeSpace,
		spaceID,
		key,
		out,
	)
}

// SpaceMap maps all available settings using the provided handlers for the given space.
func (s *Service) SpaceMap(
	ctx context.Context,
	spaceID int64,
	handlers ...SettingHandler,
) error {
	return s.Map(
		ctx,
		enum.SettingsScopeSpace,
		spaceID,
		handlers...,
	)
}

// SpaceMapWithDefaults returns default settings hydrated by space-specific values.
func SpaceMapWithDefaults[T any](
	ctx context.Context,
	service *Service,
	spaceID int64,
	getDefaults func() T,
	getMappings func(T) []SettingHandler,
) (T, error) {
	out := getDefaults()
	err := service.SpaceMap(ctx, spaceID, getMappings(out)...)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("failed to map settings: %w", err)
	}

	return out, nil
}

// SpaceUpdateWithDefaults updates space settings and returns old and new full settings snapshots.
func SpaceUpdateWithDefaults[T any](
	ctx context.Context,
	service *Service,
	spaceID int64,
	getDefaults func() T,
	getMappings func(T) []SettingHandler,
	keyValues ...KeyValue,
) (T, T, error) {
	old, err := SpaceMapWithDefaults(ctx, service, spaceID, getDefaults, getMappings)
	if err != nil {
		var zero T
		return zero, zero, fmt.Errorf("failed to map settings (old): %w", err)
	}

	err = service.SpaceSetMany(ctx, spaceID, keyValues...)
	if err != nil {
		var zero T
		return zero, zero, fmt.Errorf("failed to set settings: %w", err)
	}

	out, err := SpaceMapWithDefaults(ctx, service, spaceID, getDefaults, getMappings)
	if err != nil {
		var zero T
		return zero, zero, fmt.Errorf("failed to map settings (new): %w", err)
	}

	return old, out, nil
}

// SpaceGetDefaultBranchRecursive resolves default branch from a space, then walks up its parents.
// It returns the first configured value or falls back to the global default.
func (s *Service) SpaceGetDefaultBranchRecursive(
	ctx context.Context,
	spaceID int64,
	parentID int64,
) (string, error) {
	currentID := spaceID
	currentParentID := parentID

	for {
		var defaultBranch string
		found, err := s.SpaceGet(ctx, currentID, DefaultBranchKey, &defaultBranch)
		if err != nil {
			return "", fmt.Errorf("failed to find default branch setting for space ID %d: %w", currentID, err)
		}

		if found {
			return defaultBranch, nil
		}

		if currentParentID <= 0 {
			break
		}

		parent, err := s.spaceFinder.FindByID(ctx, currentParentID)
		if err != nil {
			return "", fmt.Errorf("failed to find parent space with id %d: %w", currentParentID, err)
		}

		currentID = parent.ID
		currentParentID = parent.ParentID
	}

	return DefaultBranch, nil
}
