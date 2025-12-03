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

package enum

// SettingsScope defines the different scopes of a setting.
type SettingsScope string

func (SettingsScope) Enum() []interface{} {
	return toInterfaceSlice(GetAllSettingsScopes())
}

var (
	// SettingsScopeSpace defines settings stored on a space level.
	SettingsScopeSpace SettingsScope = "space"

	// SettingsScopeRepo defines settings stored on a repo level.
	SettingsScopeRepo SettingsScope = "repo"

	// SettingsScopeSystem defines settings stored on a system.
	SettingsScopeSystem SettingsScope = "system"
)

func GetAllSettingsScopes() []SettingsScope {
	return []SettingsScope{
		SettingsScopeSpace,
		SettingsScopeRepo,
		SettingsScopeSystem,
	}
}
