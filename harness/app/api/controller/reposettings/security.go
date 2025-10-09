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

// SecuritySettings represents the security related part of repository settings as exposed externally.
type SecuritySettings struct {
	SecretScanningEnabled   *bool `json:"secret_scanning_enabled" yaml:"secret_scanning_enabled"`
	PrincipalCommitterMatch *bool `json:"principal_committer_match" yaml:"principal_committer_match"`
}

func GetDefaultSecuritySettings() *SecuritySettings {
	return &SecuritySettings{
		SecretScanningEnabled:   ptr.Bool(settings.DefaultSecretScanningEnabled),
		PrincipalCommitterMatch: ptr.Bool(settings.DefaultPrincipalCommitterMatch),
	}
}

func GetSecuritySettingsMappings(s *SecuritySettings) []settings.SettingHandler {
	return []settings.SettingHandler{
		settings.Mapping(settings.KeySecretScanningEnabled, s.SecretScanningEnabled),
		settings.Mapping(settings.KeyPrincipalCommitterMatch, s.PrincipalCommitterMatch),
	}
}

func GetSecuritySettingsAsKeyValues(s *SecuritySettings) []settings.KeyValue {
	kvs := make([]settings.KeyValue, 0, 2)

	if s.SecretScanningEnabled != nil {
		kvs = append(kvs, settings.KeyValue{Key: settings.KeySecretScanningEnabled, Value: *s.SecretScanningEnabled})
	}

	if s.PrincipalCommitterMatch != nil {
		kvs = append(kvs, settings.KeyValue{
			Key:   settings.KeyPrincipalCommitterMatch,
			Value: s.PrincipalCommitterMatch,
		})
	}

	return kvs
}
