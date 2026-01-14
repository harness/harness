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

package reposettings

import (
	"github.com/harness/gitness/app/services/settings"

	"github.com/gotidy/ptr"
)

// GeneralSettings represent the general repository settings as exposed externally.
type GeneralSettings struct {
	FileSizeLimit *int64 `json:"file_size_limit" yaml:"file_size_limit" description:"file size limit in bytes"`
	GitLFSEnabled *bool  `json:"git_lfs_enabled" yaml:"git_lfs_enabled"`
}

func GetDefaultGeneralSettings() *GeneralSettings {
	return &GeneralSettings{
		FileSizeLimit: ptr.Int64(settings.DefaultFileSizeLimit),
		GitLFSEnabled: ptr.Bool(settings.DefaultGitLFSEnabled),
	}
}

func GetGeneralSettingsMappings(s *GeneralSettings) []settings.SettingHandler {
	return []settings.SettingHandler{
		settings.Mapping(settings.KeyFileSizeLimit, s.FileSizeLimit),
		settings.Mapping(settings.KeyGitLFSEnabled, s.GitLFSEnabled),
	}
}

func GetGeneralSettingsAsKeyValues(s *GeneralSettings) []settings.KeyValue {
	kvs := make([]settings.KeyValue, 0, 1)

	if s.FileSizeLimit != nil {
		kvs = append(kvs, settings.KeyValue{
			Key:   settings.KeyFileSizeLimit,
			Value: s.FileSizeLimit,
		})
	}

	if s.GitLFSEnabled != nil {
		kvs = append(kvs, settings.KeyValue{
			Key:   settings.KeyGitLFSEnabled,
			Value: s.GitLFSEnabled,
		})
	}

	return kvs
}
