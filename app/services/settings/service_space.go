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
// If inherited is true, walks up the parent chain to find the first configured value.
// If inherited is false, returns only the space-specific value.
// Returns (found, error). If not found and no error, the caller should apply a default.
func (s *Service) SpaceGet(
	ctx context.Context,
	spaceID int64,
	key Key,
	inherited bool,
	out any,
) (bool, error) {
	if !inherited {
		return s.Get(
			ctx,
			enum.SettingsScopeSpace,
			spaceID,
			key,
			out,
		)
	}

	currentID := spaceID
	for currentID > 0 {
		found, err := s.Get(ctx, enum.SettingsScopeSpace, currentID, key, out)
		if err != nil {
			return false, fmt.Errorf("failed to find setting %s for space ID %d: %w", key, currentID, err)
		}

		if found {
			return true, nil
		}

		space, err := s.spaceFinder.FindByID(ctx, currentID)
		if err != nil {
			return false, fmt.Errorf("failed to find space with id %d: %w", currentID, err)
		}

		currentID = space.ParentID
	}

	return false, nil
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

// SpaceGetDefaultBranch resolves the default branch for a space.
// If inherited is true, walks up the parent chain and returns the first configured value,
// falling back to the global default if no ancestor has a value set.
// If inherited is false, returns only the space-local value,
// or the global default if no local value is set (parent settings are not consulted).
func (s *Service) SpaceGetDefaultBranch(
	ctx context.Context,
	spaceID int64,
	inherited bool,
) (string, error) {
	var defaultBranch string
	found, err := s.SpaceGet(ctx, spaceID, DefaultBranchKey, inherited, &defaultBranch)
	if err != nil {
		return "", err
	}
	if !found {
		return DefaultBranch, nil
	}
	return defaultBranch, nil
}
