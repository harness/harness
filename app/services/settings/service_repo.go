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

// RepoSet sets the value of the setting with the given key for the given repo.
func (s *Service) RepoSet(
	ctx context.Context,
	repoID int64,
	key Key,
	value any,
) error {
	return s.Set(
		ctx,
		enum.SettingsScopeRepo,
		repoID,
		key,
		value,
	)
}

// RepoSetMany sets the value of the settings with the given keys for the given repo.
func (s *Service) RepoSetMany(
	ctx context.Context,
	repoID int64,
	keyValues ...KeyValue,
) error {
	return s.SetMany(
		ctx,
		enum.SettingsScopeRepo,
		repoID,
		keyValues...,
	)
}

// RepoGet returns the value of the setting with the given key for the given repo.
func (s *Service) RepoGet(
	ctx context.Context,
	repoID int64,
	key Key,
	out any,
) (bool, error) {
	return s.Get(
		ctx,
		enum.SettingsScopeRepo,
		repoID,
		key,
		out,
	)
}

// RepoMap maps all available settings using the provided handlers for the given repo.
func (s *Service) RepoMap(
	ctx context.Context,
	repoID int64,
	handlers ...SettingHandler,
) error {
	return s.Map(
		ctx,
		enum.SettingsScopeRepo,
		repoID,
		handlers...,
	)
}

// RepoMapWithDefaults returns default settings hydrated by repo-specific values.
func RepoMapWithDefaults[T any](
	ctx context.Context,
	service *Service,
	repoID int64,
	getDefaults func() T,
	getMappings func(T) []SettingHandler,
) (T, error) {
	out := getDefaults()
	err := service.RepoMap(ctx, repoID, getMappings(out)...)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("failed to map settings: %w", err)
	}

	return out, nil
}

// RepoUpdateWithDefaults updates repo settings and returns old and new full settings snapshots.
func RepoUpdateWithDefaults[T any](
	ctx context.Context,
	service *Service,
	repoID int64,
	getDefaults func() T,
	getMappings func(T) []SettingHandler,
	keyValues ...KeyValue,
) (T, T, error) {
	old, err := RepoMapWithDefaults(ctx, service, repoID, getDefaults, getMappings)
	if err != nil {
		var zero T
		return zero, zero, fmt.Errorf("failed to map settings (old): %w", err)
	}

	err = service.RepoSetMany(ctx, repoID, keyValues...)
	if err != nil {
		var zero T
		return zero, zero, fmt.Errorf("failed to set settings: %w", err)
	}

	out, err := RepoMapWithDefaults(ctx, service, repoID, getDefaults, getMappings)
	if err != nil {
		var zero T
		return zero, zero, fmt.Errorf("failed to map settings (new): %w", err)
	}

	return old, out, nil
}
