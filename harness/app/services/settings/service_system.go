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

	"github.com/harness/gitness/types/enum"
)

// SystemSet sets the value of the setting with the given key for the system.
func (s *Service) SystemSet(
	ctx context.Context,
	key Key,
	value any,
) error {
	return s.Set(
		ctx,
		enum.SettingsScopeSystem,
		0,
		key,
		value,
	)
}

// SystemSetMany sets the values of the settings with the given keys for the system.
func (s *Service) SystemSetMany(
	ctx context.Context,
	keyValues ...KeyValue,
) error {
	return s.SetMany(
		ctx,
		enum.SettingsScopeSystem,
		0,
		keyValues...,
	)
}

// SystemGet returns the value of the setting with the given key for the system.
func (s *Service) SystemGet(
	ctx context.Context,
	key Key,
	out any,
) (bool, error) {
	return s.Get(
		ctx,
		enum.SettingsScopeSystem,
		0,
		key,
		out,
	)
}

// SystemMap maps all available settings using the provided handlers for the system.
func (s *Service) SystemMap(
	ctx context.Context,
	handlers ...SettingHandler,
) error {
	return s.Map(
		ctx,
		enum.SettingsScopeSystem,
		0,
		handlers...,
	)
}
