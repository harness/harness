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

package system

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/services/settings"

	"github.com/gotidy/ptr"
)

type Settings struct {
	InstallID *string `json:"install_id" yaml:"install_id"`
}

func getDefaultSystemSettings() *Settings {
	return &Settings{
		InstallID: ptr.String(settings.DefaultInstallID),
	}
}

func getSystemSettingsMappings(s *Settings) []settings.SettingHandler {
	return []settings.SettingHandler{
		settings.Mapping(settings.KeyInstallID, s.InstallID),
	}
}

func getSystemSettingsAsKeyValues(s *Settings) []settings.KeyValue {
	kvs := make([]settings.KeyValue, 0, 1)

	if s.InstallID != nil {
		kvs = append(kvs, settings.KeyValue{
			Key:   settings.KeyInstallID,
			Value: s.InstallID,
		})
	}
	return kvs
}

// Find returns the settings of the system.
func (s *Service) Find(
	ctx context.Context,
) (*Settings, error) {
	out := getDefaultSystemSettings()
	mappings := getSystemSettingsMappings(out)
	err := s.settings.SystemMap(ctx, mappings...)
	if err != nil {
		return nil, fmt.Errorf("failed to map settings: %w", err)
	}

	return out, nil
}

// Update updates the settings of the system.
func (s *Service) Update(
	ctx context.Context,
	in *Settings,
) (*Settings, error) {
	// read old settings values
	old := getDefaultSystemSettings()
	oldMappings := getSystemSettingsMappings(old)
	err := s.settings.SystemMap(ctx, oldMappings...)
	if err != nil {
		return nil, fmt.Errorf("failed to map settings (old): %w", err)
	}

	err = s.settings.SystemSetMany(ctx, getSystemSettingsAsKeyValues(in)...)
	if err != nil {
		return nil, fmt.Errorf("failed to set settings: %w", err)
	}

	// read all settings and return complete config
	out := getDefaultSystemSettings()
	mappings := getSystemSettingsMappings(out)
	err = s.settings.SystemMap(ctx, mappings...)
	if err != nil {
		return nil, fmt.Errorf("failed to map settings: %w", err)
	}

	return out, nil
}
